package mailer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/airtrack/pkg/zlib"
	"github.com/afk11/mail"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

type MailSender interface {
	Queue(msg db.EmailJob) error
}

type Mailer struct {
	database  db.Database
	from      string
	dialer    *mail.Dialer
	queued    []db.EmailJob
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
func encodeJob(job *db.EmailJob) ([]byte, error) {
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
	return compressed, nil
}
func decodeJob(compressed []byte) (db.EmailJob, error) {
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
	return m.database.Transaction(func(tx *sqlx.Tx) error {
		for _, job := range queued {
			ciphertext, err := encodeJob(&job)
			if err != nil {
				return err
			}
			_, err = m.database.CreateEmailJobTx(tx, now, ciphertext)
			if err != nil {
				return err
			}
		}
		return nil
	})
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

	records, err := m.database.GetPendingEmailJobs(time.Now())
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
		job, err := decodeJob(record.Job)
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
			err = m.database.Transaction(func(tx *sqlx.Tx) error {
				for _, email := range finished {
					_, err = m.database.DeleteCompletedEmail(tx, email)
					if err != nil {
						return errors.Wrapf(err, "deleting completed email %d", email.Id)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		if len(failed) > 0 {
			err = m.database.Transaction(func(tx *sqlx.Tx) error {
				for _, email := range failed {
					if email.Retries == 4 {
						_, err = m.database.MarkEmailFailedTx(tx, email)
						if err != nil {
							return errors.Wrapf(err, "marking email failed %d", email.Id)
						}
					} else {
						_, err = m.database.RetryEmailAfter(tx, email, time.Now().Add(time.Minute*2))
						if err != nil {
							return errors.Wrapf(err, "updating email retry information %d", email.Id)
						}
					}
				}
				return nil
			})
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
func NewMailer(database db.Database, from string, dialer *mail.Dialer) *Mailer {
	return &Mailer{
		database: database,
		dialer:   dialer,
		from:     from,
		queued:   make([]db.EmailJob, 0),
	}
}
