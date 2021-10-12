// 源于quickfix,增加支持表配置
package store

import (
	"database/sql"
	"fmt"
	"github.com/quickfixgo/quickfix"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/quickfixgo/quickfix/config"
)

const (
	SQLStoreMessagesTableName string = "SQLStoreMessagesTableName"
	SQLStoreSessionsTableName string = "SQLStoreSessionsTableName"
	SQLStoreLogIncomingTable  string = "SQLStoreLogIncomingTable"
	SQLStoreLogOutgoingTable  string = "SQLStoreLogOutgoingTable"
	SQLStoreLogEventTable     string = "SQLStoreLogEventTable"
	SQLStoreMaxIdleConns      string = "SQLStoreMaxIdleConns"
	SQLStoreMaxOpenConns      string = "SQLStoreMaxOpenConns"
)

type sqlStoreFactory struct {
	settings *quickfix.Settings
}

type sqlStore struct {
	sessionID          quickfix.SessionID
	cache              *memoryStore
	sqlDriver          string
	sqlDataSourceName  string
	sqlConnMaxLifetime time.Duration
	sqlMaxIdleConns    int
	sqlMaxOpenConns    int
	db                 *sql.DB
	placeholder        placeholderFunc
	messagesTableName  string
	sessionsTableName  string
	logIncomingTable   string
	logOutgoingTable   string
	logEventTable      string
}

type placeholderFunc func(int) string

var rePlaceholder = regexp.MustCompile(`\?`)

func sqlString(raw string, placeholder placeholderFunc) string {
	if placeholder == nil {
		return raw
	}
	idx := 0
	return rePlaceholder.ReplaceAllStringFunc(raw, func(s string) string {
		new := placeholder(idx)
		idx += 1
		return new
	})
}

func postgresPlaceholder(i int) string {
	return fmt.Sprintf("$%d", i+1)
}

// NewSQLStoreFactory returns a sql-based implementation of MessageStoreFactory
func NewSQLStoreFactory(settings *quickfix.Settings) quickfix.MessageStoreFactory {
	return sqlStoreFactory{settings: settings}
}

// Create creates a new SQLStore implementation of the MessageStore interface
func (f sqlStoreFactory) Create(sessionID quickfix.SessionID) (msgStore quickfix.MessageStore, err error) {
	sessionSettings, ok := f.settings.SessionSettings()[sessionID]
	if !ok {
		return nil, fmt.Errorf("unknown session: %v", sessionID)
	}
	sqlDriver, err := sessionSettings.Setting(config.SQLStoreDriver)
	if err != nil {
		return nil, err
	}
	sqlDataSourceName, err := sessionSettings.Setting(config.SQLStoreDataSourceName)
	if err != nil {
		return nil, err
	}
	sqlConnMaxLifetime := 0 * time.Second
	if sessionSettings.HasSetting(config.SQLStoreConnMaxLifetime) {
		sqlConnMaxLifetime, err = sessionSettings.DurationSetting(config.SQLStoreConnMaxLifetime)
		if err != nil {
			return nil, err
		}
	}
	sqlMaxOpenConns := 32
	if sessionSettings.HasSetting(SQLStoreMaxOpenConns) {
		sqlMaxOpenConns, err = sessionSettings.IntSetting(SQLStoreMaxOpenConns)
		if err != nil {
			return nil, err
		}
	}
	sqlMaxIdleConns := 2
	if sessionSettings.HasSetting(SQLStoreMaxIdleConns) {
		sqlMaxIdleConns, err = sessionSettings.IntSetting(SQLStoreMaxIdleConns)
		if err != nil {
			return nil, err
		}
	}
	messagesTableName := "messages"
	if sessionSettings.HasSetting(SQLStoreMessagesTableName) {
		messagesTableName, err = sessionSettings.Setting(SQLStoreMessagesTableName)
		if err != nil {
			return nil, err
		}
	}
	sessionsTableName := "sessions"
	if sessionSettings.HasSetting(SQLStoreSessionsTableName) {
		sessionsTableName, err = sessionSettings.Setting(SQLStoreSessionsTableName)
		if err != nil {
			return nil, err
		}
	}
	logIncomingTable := "messages_log"
	if sessionSettings.HasSetting(SQLStoreLogIncomingTable) {
		logIncomingTable, err = sessionSettings.Setting(SQLStoreLogIncomingTable)
		if err != nil {
			return nil, err
		}
	}
	logOutgoingTable := "messages_log"
	if sessionSettings.HasSetting(SQLStoreLogOutgoingTable) {
		logOutgoingTable, err = sessionSettings.Setting(SQLStoreLogOutgoingTable)
		if err != nil {
			return nil, err
		}
	}
	logEventTable := "event_log"
	if sessionSettings.HasSetting(SQLStoreLogEventTable) {
		logEventTable, err = sessionSettings.Setting(SQLStoreLogEventTable)
		if err != nil {
			return nil, err
		}
	}
	return newSQLStore(sessionID, sqlDriver, sqlDataSourceName, sqlConnMaxLifetime, sqlMaxOpenConns, sqlMaxIdleConns,
		messagesTableName, sessionsTableName, logIncomingTable, logOutgoingTable, logEventTable)
}

func newSQLStore(sessionID quickfix.SessionID, driver string, dataSourceName string, connMaxLifetime time.Duration,
	sqlMaxOpenConns, sqlMaxIdleConns int,
	messagesTableName, sessionsTableName, logIncomingTable, logOutgoingTable, logEventTable string) (store *sqlStore, err error) {
	store = &sqlStore{
		sessionID:          sessionID,
		cache:              &memoryStore{},
		sqlDriver:          driver,
		sqlDataSourceName:  dataSourceName,
		sqlConnMaxLifetime: connMaxLifetime,
		sqlMaxOpenConns:    sqlMaxOpenConns,
		sqlMaxIdleConns:    sqlMaxIdleConns,
		messagesTableName:  messagesTableName,
		sessionsTableName:  sessionsTableName,
		logIncomingTable:   logIncomingTable,
		logOutgoingTable:   logOutgoingTable,
		logEventTable:      logEventTable,
	}
	if err = store.cache.Reset(); err != nil {
		err = errors.Wrap(err, "cache reset")
		return
	}

	if store.sqlDriver == "postgres" {
		store.placeholder = postgresPlaceholder
	}

	if store.db, err = sql.Open(store.sqlDriver, store.sqlDataSourceName); err != nil {
		return nil, err
	}
	store.db.SetConnMaxLifetime(store.sqlConnMaxLifetime)
	store.db.SetMaxIdleConns(store.sqlMaxIdleConns)
	store.db.SetMaxOpenConns(store.sqlMaxOpenConns)

	if err = store.db.Ping(); err != nil { // ensure immediate connection
		return nil, err
	}
	if err = store.populateCache(); err != nil {
		return nil, err
	}

	return store, nil
}

// Reset deletes the store records and sets the seqnums back to 1
func (store *sqlStore) Reset() error {
	s := store.sessionID
	_, err := store.db.Exec(sqlString(`DELETE FROM `+store.messagesTableName+`
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?`, store.placeholder),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)
	if err != nil {
		return err
	}

	if err = store.cache.Reset(); err != nil {
		return err
	}

	_, err = store.db.Exec(sqlString(`UPDATE `+store.sessionsTableName+`
		SET creation_time=?, incoming_seqnum=?, outgoing_seqnum=?
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?`, store.placeholder),
		store.cache.CreationTime(), store.cache.NextTargetMsgSeqNum(), store.cache.NextSenderMsgSeqNum(),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)

	return err
}

// Refresh reloads the store from the database
func (store *sqlStore) Refresh() error {
	if err := store.cache.Reset(); err != nil {
		return err
	}
	return store.populateCache()
}

func (store *sqlStore) populateCache() error {
	s := store.sessionID
	var creationTime time.Time
	var incomingSeqNum, outgoingSeqNum int
	row := store.db.QueryRow(sqlString(`SELECT creation_time, incoming_seqnum, outgoing_seqnum
	  FROM `+store.sessionsTableName+`
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?`, store.placeholder),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)

	err := row.Scan(&creationTime, &incomingSeqNum, &outgoingSeqNum)

	// session record found, load it
	if err == nil {
		store.cache.creationTime = creationTime
		if err = store.cache.SetNextTargetMsgSeqNum(incomingSeqNum); err != nil {
			return errors.Wrap(err, "cache set next target")
		}
		if err = store.cache.SetNextSenderMsgSeqNum(outgoingSeqNum); err != nil {
			return errors.Wrap(err, "cache set next sender")
		}
		return nil
	}

	// fatal error, give up
	if err != sql.ErrNoRows {
		return err
	}

	// session record not found, create it
	_, err = store.db.Exec(sqlString(`INSERT INTO `+store.sessionsTableName+` (
			creation_time, incoming_seqnum, outgoing_seqnum,
			beginstring, session_qualifier,
			sendercompid, sendersubid, senderlocid,
			targetcompid, targetsubid, targetlocid)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, store.placeholder),
		store.cache.creationTime,
		store.cache.NextTargetMsgSeqNum(),
		store.cache.NextSenderMsgSeqNum(),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)

	return err
}

// NextSenderMsgSeqNum returns the next MsgSeqNum that will be sent
func (store *sqlStore) NextSenderMsgSeqNum() int {
	return store.cache.NextSenderMsgSeqNum()
}

// NextTargetMsgSeqNum returns the next MsgSeqNum that should be received
func (store *sqlStore) NextTargetMsgSeqNum() int {
	return store.cache.NextTargetMsgSeqNum()
}

// SetNextSenderMsgSeqNum sets the next MsgSeqNum that will be sent
func (store *sqlStore) SetNextSenderMsgSeqNum(next int) error {
	s := store.sessionID
	_, err := store.db.Exec(sqlString(`UPDATE `+store.sessionsTableName+` SET outgoing_seqnum = ?
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?`, store.placeholder),
		next, s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)
	if err != nil {
		return err
	}
	return store.cache.SetNextSenderMsgSeqNum(next)
}

// SetNextTargetMsgSeqNum sets the next MsgSeqNum that should be received
func (store *sqlStore) SetNextTargetMsgSeqNum(next int) error {
	s := store.sessionID
	_, err := store.db.Exec(sqlString(`UPDATE `+store.sessionsTableName+` SET incoming_seqnum = ?
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?`, store.placeholder),
		next, s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)
	if err != nil {
		return err
	}
	return store.cache.SetNextTargetMsgSeqNum(next)
}

// IncrNextSenderMsgSeqNum increments the next MsgSeqNum that will be sent
func (store *sqlStore) IncrNextSenderMsgSeqNum() error {
	if err := store.cache.IncrNextSenderMsgSeqNum(); err != nil {
		return errors.Wrap(err, "cache incr next")
	}
	return store.SetNextSenderMsgSeqNum(store.cache.NextSenderMsgSeqNum())
}

// IncrNextTargetMsgSeqNum increments the next MsgSeqNum that should be received
func (store *sqlStore) IncrNextTargetMsgSeqNum() error {
	if err := store.cache.IncrNextTargetMsgSeqNum(); err != nil {
		return errors.Wrap(err, "cache incr next")
	}
	return store.SetNextTargetMsgSeqNum(store.cache.NextTargetMsgSeqNum())
}

// CreationTime returns the creation time of the store
func (store *sqlStore) CreationTime() time.Time {
	return store.cache.CreationTime()
}

func (store *sqlStore) SaveMessage(seqNum int, msg []byte) error {
	s := store.sessionID

	_, err := store.db.Exec(sqlString(`INSERT INTO `+store.messagesTableName+` (
			msgseqnum, message,
			beginstring, session_qualifier,
			sendercompid, sendersubid, senderlocid,
			targetcompid, targetsubid, targetlocid)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, store.placeholder),
		seqNum, string(msg),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID)

	return err
}

func (store *sqlStore) GetMessages(beginSeqNum, endSeqNum int) ([][]byte, error) {
	s := store.sessionID
	var msgs [][]byte
	rows, err := store.db.Query(sqlString(`SELECT message FROM `+store.messagesTableName+`
		WHERE beginstring=? AND session_qualifier=?
		AND sendercompid=? AND sendersubid=? AND senderlocid=?
		AND targetcompid=? AND targetsubid=? AND targetlocid=?
		AND msgseqnum>=? AND msgseqnum<=?
		ORDER BY msgseqnum`, store.placeholder),
		s.BeginString, s.Qualifier,
		s.SenderCompID, s.SenderSubID, s.SenderLocationID,
		s.TargetCompID, s.TargetSubID, s.TargetLocationID,
		beginSeqNum, endSeqNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil, err
		}
		msgs = append(msgs, []byte(message))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return msgs, nil
}

// Close closes the store's database connection
func (store *sqlStore) Close() error {
	if store.db != nil {
		store.db.Close()
		store.db = nil
	}
	return nil
}
