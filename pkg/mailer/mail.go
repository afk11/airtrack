package mailer

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/zlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/afk11/mail"
	"io"
	"strings"
	"sync"
	"time"
)

type MailSender interface {
	Queue(msg db.EmailJob) error
}

type Mailer struct {
	dbConn    *sqlx.DB
	from      string
	key       []byte
	dialer    *mail.Dialer
	queued    []db.EmailJob
	aesgcm    cipher.AEAD
	canceller func()
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

func (m *Mailer) Queue(msg db.EmailJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queued = append(m.queued, msg)
	return nil
}
func encodeJob(aes cipher.AEAD, job *db.EmailJob) ([]byte, error) {
	// make json
	raw, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	// compress
	compressed, err := zlib.Encode(raw)
	if err != nil {
		return nil, errors.Wrap(err, "zlib decode email")
	}
	// encrypt
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, aes.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aes.Seal(nil, nonce, compressed, nil)
	ciphertext = append(ciphertext, nonce...)
	return ciphertext, nil
}
func decodeJob(aes cipher.AEAD, jobBytes []byte) (db.EmailJob, error) {
	ctLen := len(jobBytes)
	if ctLen < aes.NonceSize() {
		return db.EmailJob{}, errors.New("ciphertext too small")
	}
	// decrypt
	nonce := jobBytes[ctLen-aes.NonceSize():]
	ct := jobBytes[0 : ctLen-aes.NonceSize()]
	compressed, err := aes.Open(nil, nonce, ct, nil)
	if err != nil {
		return db.EmailJob{}, err
	}
	// decompress
	raw, err := zlib.Decode(compressed)
	if err != nil {
		return db.EmailJob{}, errors.Wrap(err, "zlib decode email")
	}
	// parse json
	job := db.EmailJob{}
	err = json.Unmarshal(raw, &job)
	if err != nil {
		return db.EmailJob{}, err
	}
	return job, nil
}
func (m *Mailer) addMailsToDb(now time.Time, queued []db.EmailJob) error {
	return db.NewTxExecer(m.dbConn, func(tx *sql.Tx) error {
		for _, job := range queued {
			ciphertext, err := encodeJob(m.aesgcm, &job)
			if err != nil {
				return err
			}
			_, err = db.CreateEmailJobTx(tx, now, ciphertext)
			if err != nil {
				return err
			}
		}
		return nil
	}).Exec()
}
func (m *Mailer) processMails() error {
	start := time.Now()
	cancelled := make(chan bool)
	defer func() {
		cancelled <- true
	}()

	go func() {
		select {
		case <-time.After(time.Minute):
			panic(fmt.Errorf("mailer running after 1 minute"))
		case <-cancelled:
			log.Debugf("mailer terminated normally after %s", time.Since(start))
			break
		}
	}()
	m.mu.Lock()
	queued := m.queued
	m.queued = nil
	m.mu.Unlock()

	if len(queued) > 0 {
		err := m.addMailsToDb(time.Now(), queued)
		if err != nil {
			return errors.Wrapf(err, "add queued emails to database")
		}
	}

	records, err := db.GetPendingEmailJobs(m.dbConn, time.Now())
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}
	log.Debugf("mailer: processing %d jobs", len(records))

	jobs := make([]db.EmailJob, 0, len(records))
	for _, record := range records {
		job, err := decodeJob(m.aesgcm, record.Job)
		if err != nil {
			return errors.Wrapf(err, "decoding job")
		}
		jobs = append(jobs, job)
	}

	if len(jobs) > 0 {

		sendCloser, err := m.dialer.Dial()
		if err != nil {
			log.Warnf("failed to connect to SMTP server: %s", err.Error())
			return err
		}

		failed := make([]db.Email, 0)
		finished := make([]db.Email, 0)
		for i, job := range jobs {
			msg := mail.NewMessage()
			msg.SetHeader("Sender", m.from)
			msg.SetHeader("To", job.To, job.To)
			msg.SetHeader("Subject", job.Subject)
			msg.SetBody("text/html", job.Body)
			for _, attach := range job.Attachments {
				msg.AttachReader(attach.ContentType, strings.NewReader(attach.Contents), mail.Rename(attach.FileName))
			}
			err := mail.Send(sendCloser, msg)
			if err != nil {
				log.Warnf("failed to send email: %s", err.Error())
				failed = append(failed, records[i])
			} else {
				finished = append(finished, records[i])
			}
		}

		err = sendCloser.Close()
		if err != nil {
			log.Warnf("failed to close SMTP dialer")
		}

		if len(finished) > 0 {
			err = db.NewTxExecer(m.dbConn, func(tx *sql.Tx) error {
				for _, email := range finished {
					_, err = db.DeleteCompletedEmail(tx, email)
					if err != nil {
						return errors.Wrapf(err, "deleting completed email %d", email.Id)
					}
				}
				return nil
			}).Exec()
			if err != nil {
				return err
			}
		}

		if len(failed) > 0 {
			err = db.NewTxExecer(m.dbConn, func(tx *sql.Tx) error {
				for _, email := range failed {
					if email.Retries == 4 {
						_, err = db.MarkEmailFailedTx(tx, email)
						if err != nil {
							return errors.Wrapf(err, "marking email failed %d", email.Id)
						}
					} else {
						_, err = db.RetryEmailAfter(tx, email, time.Now().Add(time.Minute*2))
						if err != nil {
							return errors.Wrapf(err, "updating email retry information %d", email.Id)
						}
					}
				}
				return nil
			}).Exec()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *Mailer) Start() {
	ctx, canceller := context.WithCancel(context.Background())
	m.canceller = canceller
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		normalDelay := time.Second * 5
		delay := normalDelay
		for {
			select {
			case <-time.After(delay):
				err := m.processMails()
				if err != nil {
					delay = time.Minute
				} else {
					delay = normalDelay
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (m *Mailer) Stop() {
	m.canceller()
	m.wg.Wait()
	err := m.processMails()
	if err != nil {
		panic(err)
	}
}
func NewMailer(dbConn *sqlx.DB, from string, dialer *mail.Dialer, aesgcm cipher.AEAD) *Mailer {
	return &Mailer{
		dbConn: dbConn,
		dialer: dialer,
		from:   from,
		aesgcm: aesgcm,
		queued: make([]db.EmailJob, 0),
	}
}
