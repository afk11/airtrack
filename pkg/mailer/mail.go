package mailer

import (
	"bytes"
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
	"sync"
	"time"
)

// MailSender - public interface for queuing emails to be sent.
type MailSender interface {
	Queue(msg db.EmailJob) error
}

// Mailer manages background services for sending email.
// New emails are queued in queued until the processing
// coroutine saves them to the database in a batch. The
// processing routine also searches for new emails to send
// Implements MailSender
type Mailer struct {
	database  db.Database
	from      string
	dialer    *mail.Dialer
	queued    []db.EmailJob
	canceller func()
	mu        sync.RWMutex
	wg        sync.WaitGroup
}

// NewMailer creates a new Mailer
func NewMailer(database db.Database, from string, dialer *mail.Dialer) *Mailer {
	return &Mailer{
		database: database,
		dialer:   dialer,
		from:     from,
		queued:   make([]db.EmailJob, 0),
	}
}

// Queue adds job to the queue so it can be persisted later.
// See MailSender.Queue
func (m *Mailer) Queue(job db.EmailJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queued = append(m.queued, job)
	return nil
}

// Start invokes the processing goroutine
func (m *Mailer) Start() {
	ctx, canceller := context.WithCancel(context.Background())
	m.canceller = canceller
	m.wg.Add(1)
	go m.periodicallyProcessMails(ctx)
}

// periodicallyProcessEmails runs in a loop until the shutdown
// signal is received. In each iteration it calls processMails.
// If errors arise, an extra delay is used.
func (m *Mailer) periodicallyProcessMails(ctx context.Context) {
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
}

// processMails is called periodically to save persist new emails
// and to send emails that need to be sent.
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

		failedEmails := make([]db.Email, 0)
		finishedEmails := make([]db.Email, 0)
		for i := range jobs {
			msg := mail.NewMessage()
			msg.SetHeader("Sender", m.from)
			msg.SetHeader("To", jobs[i].To, jobs[i].To)
			msg.SetHeader("Subject", jobs[i].Subject)
			msg.SetBody("text/html", jobs[i].Body)
			for _, attach := range jobs[i].Attachments {
				msg.AttachReader(attach.ContentType, bytes.NewBuffer(attach.Contents), mail.Rename(attach.FileName))
			}
			err := mail.Send(sendCloser, msg)
			if err != nil {
				log.Warnf("failed to send email: %s", err.Error())
				failedEmails = append(failedEmails, records[i])
			} else {
				finishedEmails = append(finishedEmails, records[i])
			}
		}

		err = sendCloser.Close()
		if err != nil {
			log.Warnf("failed to close SMTP dialer")
		}

		if len(finishedEmails) > 0 {
			err = m.database.Transaction(func(tx *sqlx.Tx) error {
				for i := range finishedEmails {
					_, err = m.database.DeleteCompletedEmail(tx, finishedEmails[i])
					if err != nil {
						return errors.Wrapf(err, "deleting completed email %d", finishedEmails[i].Id)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		if len(failedEmails) > 0 {
			err = m.database.Transaction(func(tx *sqlx.Tx) error {
				for i := range failedEmails {
					if failedEmails[i].Retries == 4 {
						_, err = m.database.MarkEmailFailedTx(tx, failedEmails[i])
						if err != nil {
							return errors.Wrapf(err, "marking email failed %d", failedEmails[i].Id)
						}
					} else {
						_, err = m.database.RetryEmailAfter(tx, failedEmails[i], time.Now().Add(time.Minute*2))
						if err != nil {
							return errors.Wrapf(err, "updating email retry information %d", failedEmails[i].Id)
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

// addMailsToDb encodes and persist queued jobs.
func (m *Mailer) addMailsToDb(now time.Time, queued []db.EmailJob) error {
	return m.database.Transaction(func(tx *sqlx.Tx) error {
		for idx := range queued {
			encoded, err := encodeJob(&queued[idx])
			if err != nil {
				return err
			}
			_, err = m.database.CreateEmailJobTx(tx, now, encoded)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Stop cancels the processing goroutine and waits for it to finish
func (m *Mailer) Stop() {
	m.canceller()
	m.wg.Wait()
	err := m.processMails()
	if err != nil {
		panic(err)
	}
}

// encodeJob takes a job and encodes it into a compressed payload
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

// decodeJob takes a compressed job and decodes it into a db.EmailJob.
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
