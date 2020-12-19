package mailer

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/afk11/airtrack/pkg/db"
	"github.com/afk11/mail"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"sync"
	"time"
)

type (
	// EmailAttachment is a JSON structure for an email attachment in
	// encoded job.
	EmailAttachment struct {
		ContentType string `json:"content_type"`
		FileName    string `json:"filename"`
		Contents    []byte `json:"contents"`
	}
	// EmailJob - the JSON structure for db.Email Job field.
	EmailJob struct {
		To          string            `json:"to"`
		Subject     string            `json:"subject"`
		Body        string            `json:"body"`
		Attachments []EmailAttachment `json:"attachments"`
	}
	// MailSender - public interface for queuing emails to be sent.
	MailSender interface {
		Queue(msg EmailJob) error
	}
	// Mailer manages background services for sending email.
	// New emails are queued in queued until the processing
	// coroutine saves them to the database in a batch. The
	// processing routine also searches for new emails to send
	// Implements MailSender
	Mailer struct {
		database  db.Database
		from      string
		dialer    *mail.Dialer
		queued    []EmailJob
		canceller func()
		mu        sync.RWMutex
		wg        sync.WaitGroup
	}
)

// NewMailer creates a new Mailer
func NewMailer(database db.Database, from string, dialer *mail.Dialer) *Mailer {
	return &Mailer{
		database: database,
		dialer:   dialer,
		from:     from,
		queued:   make([]EmailJob, 0),
	}
}

// Queue adds job to the queue so it can be persisted later.
// See MailSender.Queue
func (m *Mailer) Queue(job EmailJob) error {
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
	normalDelay := time.Second * 30
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

	jobs := make([]EmailJob, 0, len(records))
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
			err = m.database.Transaction(func(tx db.Queries) error {
				for i := range finishedEmails {
					_, err = tx.DeleteCompletedEmail(finishedEmails[i])
					if err != nil {
						return errors.Wrapf(err, "deleting completed email %d", finishedEmails[i].ID)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		if len(failedEmails) > 0 {
			err = m.database.Transaction(func(tx db.Queries) error {
				for i := range failedEmails {
					if failedEmails[i].Retries == 4 {
						_, err = tx.MarkEmailFailed(&failedEmails[i])
						if err != nil {
							return errors.Wrapf(err, "marking email failed %d", failedEmails[i].ID)
						}
					} else {
						_, err = tx.RetryEmailAfter(&failedEmails[i], time.Now().Add(time.Minute*2))
						if err != nil {
							return errors.Wrapf(err, "updating email retry information %d", failedEmails[i].ID)
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
func (m *Mailer) addMailsToDb(now time.Time, queued []EmailJob) error {
	return m.database.Transaction(func(tx db.Queries) error {
		for idx := range queued {
			encoded, err := encodeJob(&queued[idx])
			if err != nil {
				return err
			}
			_, err = m.database.CreateEmailJob(now, encoded)
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
func encodeJob(job *EmailJob) ([]byte, error) {
	// make json
	raw, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	if _, err := w.Write(raw); err != nil {
		return nil, err
	} else if err := w.Close(); err != nil {
		return nil, err
	}

	return compressed.Bytes(), nil
}

// decodeJob takes a compressed job and decodes it into a EmailJob.
func decodeJob(compressed []byte) (EmailJob, error) {
	// decompress
	r, err := gzip.NewReader(bytes.NewBuffer(compressed))
	if err != nil {
		return EmailJob{}, errors.Wrapf(err, "creating gzip reader for kml")
	}
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return EmailJob{}, errors.Wrapf(err, "decompressing gzipped kml")
	}

	// parse json
	job := EmailJob{}
	err = json.Unmarshal(raw, &job)
	if err != nil {
		return EmailJob{}, err
	}
	return job, nil
}
