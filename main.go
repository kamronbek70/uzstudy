package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================
//  KONSTANTALAR
// ============================================================

const (
	BotToken        = "8488548469:AAGUX_aXxk41jVuOCKRHsRdbBcAe2G2vRgM"
	ApiBaseURL      = "https://api.telegram.org/bot"
	PollInterval    = 1 * time.Second
	MaxUpdateOffset = 100
	AppVersion      = "2.0.0"
	AppName         = "UzStudy Bot"
	DBFile          = "uzstudy.db"
)

// ← O'z admin ID ingizni yozing (@userinfobot orqali oling)
var AdminIDs = []int64{
	6053179118,
}

// ← O'z kanallaringizni yozing
var RequiredChannels = []ChannelInfo{
	{Username: "@UzStudyCommunity", Title: "UzStudyCommunity"},
	{Username: "@UzStudyYangiliklar", Title: "UzStudyYangiliklar"},
	{Username: "@UzStudyKanal", Title: "UzStudyKanal"},
}

// Holatlar
const (
	StateNone              = ""
	StateWaitingPhone      = "waiting_phone"
	StateVerifyChannels    = "verify_channels"
	StateWaitingNote       = "waiting_note"
	StateWaitingGoal       = "waiting_goal"
	StateWaitingWordUz     = "waiting_word_uz"
	StateWaitingWordEn     = "waiting_word_en"
	StateWaitingWordDesc   = "waiting_word_desc"
	StateWaitingPomodoro   = "waiting_pomodoro"
	StateQuizActive        = "quiz_active"
	StateWaitingFeedback   = "waiting_feedback"
	StateWaitingGoalMinute = "waiting_goal_minute"
	StateWaitingDeleteNote = "waiting_delete_note"
	StateAdminBroadcast    = "admin_broadcast"
)

const (
	CategoryMath    = "matematika"
	CategoryHistory = "tarix"
	CategoryEnglish = "ingliz"
	CategoryScience = "fizika"
	CategoryGeneral = "umumiy"
)

// ============================================================
//  TELEGRAM API TIPLARI
// ============================================================

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int      `json:"message_id"`
	From      *User    `json:"from"`
	Chat      *Chat    `json:"chat"`
	Text      string   `json:"text"`
	Date      int64    `json:"date"`
	Contact   *Contact `json:"contact"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserID      int64  `json:"user_id"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Username string `json:"username"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type GetUpdatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type SendMessageResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type ReplyKeyboardButton struct {
	Text           string `json:"text"`
	RequestContact bool   `json:"request_contact,omitempty"`
}

type ReplyKeyboardMarkup struct {
	Keyboard        [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                    `json:"resize_keyboard"`
	OneTimeKeyboard bool                    `json:"one_time_keyboard"`
}

type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

type ChatMemberResponse struct {
	OK     bool       `json:"ok"`
	Result ChatMember `json:"result"`
}

type ChatMember struct {
	Status string `json:"status"`
}

type ChannelInfo struct {
	Username string
	Title    string
}

// ============================================================
//  MA'LUMOT TIPLARI
// ============================================================

type Note struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	Priority  string    `json:"priority"`
}

type VocabWord struct {
	ID         int       `json:"id"`
	Uzbek      string    `json:"uzbek"`
	English    string    `json:"english"`
	Desc       string    `json:"desc"`
	Learned    bool      `json:"learned"`
	AddedAt    time.Time `json:"added_at"`
	LearnCount int       `json:"learn_count"`
}

type QuizQuestion struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Correct  int      `json:"correct"`
	Category string   `json:"category"`
	Points   int      `json:"points"`
	Explain  string   `json:"explain"`
}

type QuizSession struct {
	Questions    []QuizQuestion `json:"questions"`
	CurrentIndex int            `json:"current_index"`
	Score        int            `json:"score"`
	Answered     int            `json:"answered"`
	Correct      int            `json:"correct"`
	Category     string         `json:"category"`
	StartTime    time.Time      `json:"start_time"`
}

type PomodoroSession struct {
	Duration   int       `json:"duration"`
	StartTime  time.Time `json:"start_time"`
	Task       string    `json:"task"`
	IsRunning  bool      `json:"is_running"`
	CycleCount int       `json:"cycle_count"`
}

type DailyGoal struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	TargetMin   int       `json:"target_min"`
	SpentMin    int       `json:"spent_min"`
	Date        string    `json:"date"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserStats struct {
	UserID        int64     `json:"user_id"`
	FirstName     string    `json:"first_name"`
	Username      string    `json:"username"`
	TotalQuiz     int       `json:"total_quiz"`
	TotalScore    int       `json:"total_score"`
	BestScore     int       `json:"best_score"`
	TotalPomodoro int       `json:"total_pomodoro"`
	WordsAdded    int       `json:"words_added"`
	WordsLearned  int       `json:"words_learned"`
	NotesCreated  int       `json:"notes_created"`
	NotesDone     int       `json:"notes_done"`
	DaysActive    int       `json:"days_active"`
	LastActive    time.Time `json:"last_active"`
	JoinedAt      time.Time `json:"joined_at"`
	Streak        int       `json:"streak"`
	LastStreakDay string    `json:"last_streak_day"`
	TotalGoals    int       `json:"total_goals"`
	GoalsDone     int       `json:"goals_done"`
}

type UserSession struct {
	UserID        int64             `json:"user_id"`
	ChatID        int64             `json:"chat_id"`
	State         string            `json:"state"`
	Notes         []Note            `json:"notes"`
	Vocabulary    []VocabWord       `json:"vocabulary"`
	Quiz          *QuizSession      `json:"quiz"`
	Pomodoro      *PomodoroSession  `json:"pomodoro"`
	Goals         []DailyGoal       `json:"goals"`
	Stats         UserStats         `json:"stats"`
	TempData      map[string]string `json:"temp_data"`
	LastMessageID int               `json:"last_message_id"`
	Verified      bool              `json:"verified"`
	Phone         string            `json:"phone"`
	FirstName     string            `json:"first_name"`
	LastName      string            `json:"last_name"`
	Username      string            `json:"username"`
}

// ============================================================
//  SQLITE DATABASE
// ============================================================

type Database struct {
	db *sql.DB
	mu sync.Mutex
}

func NewDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path+"?_journal=WAL&_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("DB ochish xatosi: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("DB ping xatosi: %w", err)
	}
	d := &Database{db: db}
	if err = d.migrate(); err != nil {
		return nil, fmt.Errorf("Migrate xatosi: %w", err)
	}
	return d, nil
}

func (d *Database) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id    INTEGER PRIMARY KEY,
			chat_id    INTEGER NOT NULL DEFAULT 0,
			first_name TEXT    NOT NULL DEFAULT '',
			last_name  TEXT    NOT NULL DEFAULT '',
			username   TEXT    NOT NULL DEFAULT '',
			phone      TEXT    NOT NULL DEFAULT '',
			verified   INTEGER NOT NULL DEFAULT 0,
			joined_at  TEXT    NOT NULL DEFAULT '',
			last_seen  TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS user_data (
			user_id    INTEGER PRIMARY KEY,
			notes_json TEXT    NOT NULL DEFAULT '[]',
			vocab_json TEXT    NOT NULL DEFAULT '[]',
			goals_json TEXT    NOT NULL DEFAULT '[]',
			stats_json TEXT    NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS feedbacks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL DEFAULT 0,
			first_name TEXT    NOT NULL DEFAULT '',
			last_name  TEXT    NOT NULL DEFAULT '',
			username   TEXT    NOT NULL DEFAULT '',
			phone      TEXT    NOT NULL DEFAULT '',
			text       TEXT    NOT NULL DEFAULT '',
			created_at TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS pomodoro_log (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL DEFAULT 0,
			duration   INTEGER NOT NULL DEFAULT 0,
			completed  INTEGER NOT NULL DEFAULT 0,
			created_at TEXT    NOT NULL DEFAULT ''
		)`,
	}
	for _, s := range stmts {
		if _, err := d.db.Exec(s); err != nil {
			return fmt.Errorf("migrate exec xato: %w", err)
		}
	}
	return nil
}

// ---------- USER ----------

func (d *Database) UpsertUser(userID, chatID int64, firstName, lastName, username, phone string, verified bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	_, err := d.db.Exec(`
		INSERT INTO users (user_id, chat_id, first_name, last_name, username, phone, verified, joined_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			chat_id    = excluded.chat_id,
			first_name = excluded.first_name,
			last_name  = excluded.last_name,
			username   = excluded.username,
			phone      = CASE WHEN excluded.phone != '' THEN excluded.phone ELSE phone END,
			verified   = CASE WHEN excluded.verified = 1 THEN 1 ELSE verified END,
			last_seen  = excluded.last_seen`,
		userID, chatID, firstName, lastName, username, phone, boolToInt(verified), now, now)
	return err
}

func (d *Database) SetVerified(userID int64, phone string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`UPDATE users SET verified = 1, phone = ? WHERE user_id = ?`, phone, userID)
	return err
}

func (d *Database) IsVerified(userID int64) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var v int
	var phone string
	err := d.db.QueryRow(`SELECT verified, phone FROM users WHERE user_id = ?`, userID).Scan(&v, &phone)
	if err != nil {
		return false, ""
	}
	return v == 1, phone
}

func (d *Database) GetAllUsers() ([]map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.db.Query(`SELECT user_id, chat_id, first_name, last_name, username, phone, verified, joined_at, last_seen FROM users ORDER BY joined_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]string
	for rows.Next() {
		var uid, cid int64
		var fn, ln, un, ph, ja, ls string
		var vf int
		if err2 := rows.Scan(&uid, &cid, &fn, &ln, &un, &ph, &vf, &ja, &ls); err2 != nil {
			continue
		}
		m := map[string]string{
			"user_id":    strconv.FormatInt(uid, 10),
			"chat_id":    strconv.FormatInt(cid, 10),
			"first_name": fn,
			"last_name":  ln,
			"username":   un,
			"phone":      ph,
			"verified":   strconv.Itoa(vf),
			"joined_at":  ja,
		}
		result = append(result, m)
	}
	return result, nil
}

func (d *Database) CountUsers() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	var c int
	d.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&c)
	return c
}

func (d *Database) CountVerified() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	var c int
	d.db.QueryRow(`SELECT COUNT(*) FROM users WHERE verified = 1`).Scan(&c)
	return c
}

// ---------- USER DATA ----------

func (d *Database) SaveUserData(sess *UserSession) error {
	notesB, _ := json.Marshal(sess.Notes)
	vocabB, _ := json.Marshal(sess.Vocabulary)
	goalsB, _ := json.Marshal(sess.Goals)
	statsB, _ := json.Marshal(sess.Stats)
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`
		INSERT INTO user_data (user_id, notes_json, vocab_json, goals_json, stats_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			notes_json = excluded.notes_json,
			vocab_json = excluded.vocab_json,
			goals_json = excluded.goals_json,
			stats_json = excluded.stats_json`,
		sess.UserID, string(notesB), string(vocabB), string(goalsB), string(statsB))
	return err
}

func (d *Database) LoadUserData(userID int64) (notes []Note, vocab []VocabWord, goals []DailyGoal, stats UserStats) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var ns, vs, gs, ss string
	err := d.db.QueryRow(`SELECT notes_json, vocab_json, goals_json, stats_json FROM user_data WHERE user_id = ?`, userID).
		Scan(&ns, &vs, &gs, &ss)
	if err != nil {
		return []Note{}, []VocabWord{}, []DailyGoal{}, UserStats{}
	}
	json.Unmarshal([]byte(ns), &notes)
	json.Unmarshal([]byte(vs), &vocab)
	json.Unmarshal([]byte(gs), &goals)
	json.Unmarshal([]byte(ss), &stats)
	if notes == nil {
		notes = []Note{}
	}
	if vocab == nil {
		vocab = []VocabWord{}
	}
	if goals == nil {
		goals = []DailyGoal{}
	}
	return
}

// ---------- FEEDBACK ----------

func (d *Database) SaveFeedback(userID int64, firstName, lastName, username, phone, text string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec(`
		INSERT INTO feedbacks (user_id, first_name, last_name, username, phone, text, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, firstName, lastName, username, phone, text, time.Now().Format(time.RFC3339))
	return err
}

func (d *Database) GetAllFeedbacks() ([]map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.db.Query(`SELECT id, user_id, first_name, last_name, username, phone, text, created_at FROM feedbacks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]string
	for rows.Next() {
		var id, uid int64
		var fn, ln, un, ph, tx, ca string
		if err2 := rows.Scan(&id, &uid, &fn, &ln, &un, &ph, &tx, &ca); err2 != nil {
			continue
		}
		result = append(result, map[string]string{
			"id":         strconv.FormatInt(id, 10),
			"user_id":    strconv.FormatInt(uid, 10),
			"first_name": fn,
			"last_name":  ln,
			"username":   un,
			"phone":      ph,
			"text":       tx,
			"created_at": ca,
		})
	}
	return result, nil
}

func (d *Database) CountFeedbacks() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	var c int
	d.db.QueryRow(`SELECT COUNT(*) FROM feedbacks`).Scan(&c)
	return c
}

// ---------- POMODORO LOG ----------

func (d *Database) LogPomodoro(userID int64, duration int, completed bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.db.Exec(`INSERT INTO pomodoro_log (user_id, duration, completed, created_at) VALUES (?, ?, ?, ?)`,
		userID, duration, boolToInt(completed), time.Now().Format(time.RFC3339))
}

// ---------- STATS ----------

func (d *Database) GetLeaderboard() ([]UserStats, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	rows, err := d.db.Query(`SELECT stats_json FROM user_data`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []UserStats
	for rows.Next() {
		var s string
		if err2 := rows.Scan(&s); err2 != nil {
			continue
		}
		var st UserStats
		if err3 := json.Unmarshal([]byte(s), &st); err3 != nil {
			continue
		}
		all = append(all, st)
	}
	return all, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ============================================================
//  IN-MEMORY SESSION
// ============================================================

type Storage struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
	db       *Database
}

func NewStorage(db *Database) *Storage {
	return &Storage{
		sessions: make(map[int64]*UserSession),
		db:       db,
	}
}

func (s *Storage) GetSession(userID int64) *UserSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[userID]
}

func (s *Storage) setRaw(sess *UserSession) {
	s.sessions[sess.UserID] = sess
}

func (s *Storage) SetSession(sess *UserSession) {
	s.mu.Lock()
	s.sessions[sess.UserID] = sess
	s.mu.Unlock()
	go func() {
		if err := s.db.SaveUserData(sess); err != nil {
			log.Printf("SaveUserData xato user=%d: %v", sess.UserID, err)
		}
	}()
}

func (s *Storage) GetOrCreate(user *User, chatID int64) *UserSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.sessions[user.ID]; ok {
		sess.ChatID = chatID
		sess.FirstName = user.FirstName
		sess.LastName = user.LastName
		sess.Username = user.Username
		today := time.Now().Format("2006-01-02")
		if sess.Stats.LastStreakDay != today {
			yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			if sess.Stats.LastStreakDay == yesterday {
				sess.Stats.Streak++
			} else if sess.Stats.LastStreakDay != "" && sess.Stats.LastStreakDay != today {
				sess.Stats.Streak = 1
			}
			sess.Stats.LastStreakDay = today
			sess.Stats.DaysActive++
		}
		sess.Stats.LastActive = time.Now()
		return sess
	}

	// DB dan yuklab olish
	notes, vocab, goals, stats := s.db.LoadUserData(user.ID)
	verified, phone := s.db.IsVerified(user.ID)

	if stats.UserID == 0 {
		stats = UserStats{
			UserID:        user.ID,
			FirstName:     user.FirstName,
			Username:      user.Username,
			JoinedAt:      time.Now(),
			LastActive:    time.Now(),
			Streak:        1,
			LastStreakDay: time.Now().Format("2006-01-02"),
			DaysActive:    1,
		}
	} else {
		stats.FirstName = user.FirstName
		stats.Username = user.Username
		stats.LastActive = time.Now()
		today := time.Now().Format("2006-01-02")
		if stats.LastStreakDay != today {
			yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			if stats.LastStreakDay == yesterday {
				stats.Streak++
			} else if stats.LastStreakDay != "" {
				stats.Streak = 1
			}
			stats.LastStreakDay = today
			stats.DaysActive++
		}
	}

	sess := &UserSession{
		UserID:     user.ID,
		ChatID:     chatID,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Username:   user.Username,
		State:      StateNone,
		Notes:      notes,
		Vocabulary: vocab,
		Goals:      goals,
		Stats:      stats,
		TempData:   make(map[string]string),
		Verified:   verified,
		Phone:      phone,
	}
	s.sessions[user.ID] = sess
	return sess
}

func (s *Storage) AllSessions() []*UserSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*UserSession, 0, len(s.sessions))
	for _, v := range s.sessions {
		result = append(result, v)
	}
	return result
}

// ============================================================
//  QUIZ SAVOLLARI
// ============================================================

func GetAllQuestions() []QuizQuestion {
	return []QuizQuestion{
		// MATEMATIKA
		{Question: "2^10 necha?", Options: []string{"512", "1024", "2048", "256"}, Correct: 1, Category: CategoryMath, Points: 10, Explain: "2^10 = 1024"},
		{Question: "π ning taxminiy qiymati?", Options: []string{"3.14159", "2.71828", "1.61803", "1.41421"}, Correct: 0, Category: CategoryMath, Points: 10, Explain: "π ≈ 3.14159"},
		{Question: "Fibonacci 10-elementi?", Options: []string{"34", "55", "89", "44"}, Correct: 1, Category: CategoryMath, Points: 15, Explain: "1,1,2,3,5,8,13,21,34,55 → 55"},
		{Question: "x^2 = 49 bo'lsa, x = ?", Options: []string{"6", "7", "8", "9"}, Correct: 1, Category: CategoryMath, Points: 10, Explain: "√49 = 7"},
		{Question: "100 ning 15% i?", Options: []string{"10", "12", "15", "20"}, Correct: 2, Category: CategoryMath, Points: 5, Explain: "100×0.15 = 15"},
		{Question: "Uchburchak burchaklari yig'indisi?", Options: []string{"90°", "180°", "270°", "360°"}, Correct: 1, Category: CategoryMath, Points: 5, Explain: "Har doim 180°"},
		{Question: "1000 ÷ 8 = ?", Options: []string{"115", "120", "125", "130"}, Correct: 2, Category: CategoryMath, Points: 10, Explain: "1000 ÷ 8 = 125"},
		{Question: "10 gacha eng katta tub son?", Options: []string{"7", "8", "9", "10"}, Correct: 0, Category: CategoryMath, Points: 10, Explain: "7 — faqat 1 va o'ziga bo'linadi"},
		// TARIX
		{Question: "Amir Temur qaysi yili tug'ilgan?", Options: []string{"1320", "1336", "1350", "1370"}, Correct: 1, Category: CategoryHistory, Points: 10, Explain: "1336-yil, Shahrisabz"},
		{Question: "1-Jahon urushi qachon boshlangan?", Options: []string{"1912", "1914", "1916", "1918"}, Correct: 1, Category: CategoryHistory, Points: 10, Explain: "1914-yil iyulda"},
		{Question: "Ipak yo'li qayerga borgan?", Options: []string{"Xitoy—Hindiston", "Xitoy—Rim", "Arabiston—Hindiston", "Eron—Misr"}, Correct: 1, Category: CategoryHistory, Points: 15, Explain: "Xitoydan Rim imperiyasiga"},
		{Question: "O'zbekiston mustaqilligi?", Options: []string{"1989", "1990", "1991", "1992"}, Correct: 2, Category: CategoryHistory, Points: 5, Explain: "1991-yil 1-sentabr"},
		{Question: "Al-Xorazmiy asari?", Options: []string{"Kitob ul-Jabr val-Muqobala", "Qomusul-Ulum", "Al-Qonun", "Avesto"}, Correct: 0, Category: CategoryHistory, Points: 15, Explain: "Algebra fanini yaratdi"},
		{Question: "Samarqand poytaxt qachon bo'ldi?", Options: []string{"1370", "1380", "1395", "1405"}, Correct: 0, Category: CategoryHistory, Points: 10, Explain: "1370-yilda Amir Temur"},
		// INGLIZ TILI
		{Question: "'Kitob' inglizcha?", Options: []string{"Pen", "Book", "Desk", "Chair"}, Correct: 1, Category: CategoryEnglish, Points: 5, Explain: "Kitob = Book"},
		{Question: "'I am going' qaysi zamon?", Options: []string{"Simple Present", "Present Continuous", "Future Simple", "Past Continuous"}, Correct: 1, Category: CategoryEnglish, Points: 10, Explain: "to be + V-ing = Present Continuous"},
		{Question: "'Beautiful' antonimi?", Options: []string{"Pretty", "Ugly", "Cute", "Handsome"}, Correct: 1, Category: CategoryEnglish, Points: 10, Explain: "Beautiful → Ugly"},
		{Question: "She ___ to work every day.", Options: []string{"go", "goes", "going", "gone"}, Correct: 1, Category: CategoryEnglish, Points: 10, Explain: "3-shaxs birlik: -s/-es"},
		{Question: "'Run' o'tgan zamon?", Options: []string{"Runned", "Ran", "Run", "Runed"}, Correct: 1, Category: CategoryEnglish, Points: 10, Explain: "Run → Ran (irregular)"},
		{Question: "How ___ people are there?", Options: []string{"much", "many", "some", "any"}, Correct: 1, Category: CategoryEnglish, Points: 10, Explain: "Sanaladigan uchun 'many'"},
		// FIZIKA
		{Question: "Yorug'lik tezligi?", Options: []string{"299,792 km/s", "300,000 km/s", "200,000 km/s", "150,000 km/s"}, Correct: 0, Category: CategoryScience, Points: 10, Explain: "c ≈ 299,792,458 m/s"},
		{Question: "Suv qaynash harorati?", Options: []string{"90°C", "95°C", "100°C", "105°C"}, Correct: 2, Category: CategoryScience, Points: 5, Explain: "100°C (1 atm)"},
		{Question: "g = ?", Options: []string{"8.8 m/s²", "9.8 m/s²", "10.8 m/s²", "11.8 m/s²"}, Correct: 1, Category: CategoryScience, Points: 10, Explain: "g ≈ 9.8 m/s²"},
		{Question: "Atom yadrosida nima?", Options: []string{"Proton va elektron", "Proton va neytron", "Elektron va neytron", "Faqat proton"}, Correct: 1, Category: CategoryScience, Points: 10, Explain: "Proton + neytron"},
		{Question: "Ohm qonuni: U = ?", Options: []string{"I + R", "I × R", "I ÷ R", "R ÷ I"}, Correct: 1, Category: CategoryScience, Points: 10, Explain: "U = I × R"},
		{Question: "Proton massasi?", Options: []string{"1.67×10^-27 kg", "9.11×10^-31 kg", "1.00×10^-20 kg", "5.5×10^-24 kg"}, Correct: 0, Category: CategoryScience, Points: 15, Explain: "≈ 1.673 × 10^-27 kg"},
		// UMUMIY BILIM
		{Question: "Eng ko'p gapiriladigan til?", Options: []string{"Ingliz", "Ispan", "Mandarin", "Arab"}, Correct: 2, Category: CategoryGeneral, Points: 10, Explain: "Mandarin 1+ mlrd"},
		{Question: "Olimpiya qancha yilda bir?", Options: []string{"2", "3", "4", "5"}, Correct: 2, Category: CategoryGeneral, Points: 5, Explain: "Har 4 yilda"},
		{Question: "Eng katta sayyora?", Options: []string{"Saturn", "Neptun", "Yupiter", "Uran"}, Correct: 2, Category: CategoryGeneral, Points: 10, Explain: "Yupiter"},
		{Question: "Everest balandligi?", Options: []string{"7,848 m", "8,848 m", "9,848 m", "6,848 m"}, Correct: 1, Category: CategoryGeneral, Points: 5, Explain: "8,848.86 metr"},
		{Question: "Insonida necha suyak?", Options: []string{"186", "206", "226", "246"}, Correct: 1, Category: CategoryGeneral, Points: 10, Explain: "206 ta"},
		{Question: "Bir yilda necha sekund?", Options: []string{"31,536,000", "30,000,000", "25,000,000", "28,000,000"}, Correct: 0, Category: CategoryGeneral, Points: 15, Explain: "365×24×60×60"},
		{Question: "DNA to'liq nomi?", Options: []string{"Deoxyribonucleic Acid", "Diribonucleic Acid", "Dioxyribose Acid", "Dinucleic Acid"}, Correct: 0, Category: CategoryGeneral, Points: 10, Explain: "Deoxyribonucleic Acid"},
		{Question: "Python kim yaratgan?", Options: []string{"James Gosling", "Guido van Rossum", "Bjarne Stroustrup", "Dennis Ritchie"}, Correct: 1, Category: CategoryGeneral, Points: 10, Explain: "Guido van Rossum, 1991"},
	}
}

func GetQuestionsByCategory(category string) []QuizQuestion {
	all := GetAllQuestions()
	if category == "all" {
		return all
	}
	var res []QuizQuestion
	for _, q := range all {
		if q.Category == category {
			res = append(res, q)
		}
	}
	return res
}

func ShuffleQuestions(qs []QuizQuestion, n int) []QuizQuestion {
	rand.Shuffle(len(qs), func(i, j int) { qs[i], qs[j] = qs[j], qs[i] })
	if n > len(qs) {
		n = len(qs)
	}
	return qs[:n]
}

// ============================================================
//  BOT STRUCT
// ============================================================

type Bot struct {
	token   string
	baseURL string
	client  *http.Client
	storage *Storage
	db      *Database
}

func NewBot(token string, db *Database) *Bot {
	b := &Bot{
		token:   token,
		baseURL: ApiBaseURL + token,
		client:  &http.Client{Timeout: 30 * time.Second},
		storage: NewStorage(db),
		db:      db,
	}
	b.setCommands()
	return b
}

// ============================================================
//  HTTP YORDAMCHILARI
// ============================================================

func (b *Bot) doRequest(method string, params url.Values) ([]byte, error) {
	resp, err := b.client.PostForm(b.baseURL+"/"+method, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, e := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if e != nil {
			break
		}
	}
	return buf, nil
}

func (b *Bot) GetUpdates(offset int) ([]Update, error) {
	p := url.Values{}
	p.Set("offset", strconv.Itoa(offset))
	p.Set("timeout", "30")
	p.Set("limit", strconv.Itoa(MaxUpdateOffset))
	p.Set("allowed_updates", `["message","callback_query"]`)
	data, err := b.doRequest("getUpdates", p)
	if err != nil {
		return nil, err
	}
	var resp GetUpdatesResponse
	if err = json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return nil, fmt.Errorf("getUpdates xato")
	}
	return resp.Result, nil
}

func (b *Bot) SendMessage(chatID int64, text string, markup interface{}) (*Message, error) {
	p := url.Values{}
	p.Set("chat_id", strconv.FormatInt(chatID, 10))
	p.Set("text", text)
	p.Set("parse_mode", "HTML")
	if markup != nil {
		if mj, err := json.Marshal(markup); err == nil {
			p.Set("reply_markup", string(mj))
		}
	}
	data, err := b.doRequest("sendMessage", p)
	if err != nil {
		return nil, err
	}
	var resp SendMessageResponse
	if err = json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return nil, fmt.Errorf("sendMessage xato")
	}
	return &resp.Result, nil
}

func (b *Bot) EditMessage(chatID int64, msgID int, text string, markup interface{}) error {
	p := url.Values{}
	p.Set("chat_id", strconv.FormatInt(chatID, 10))
	p.Set("message_id", strconv.Itoa(msgID))
	p.Set("text", text)
	p.Set("parse_mode", "HTML")
	if markup != nil {
		if mj, err := json.Marshal(markup); err == nil {
			p.Set("reply_markup", string(mj))
		}
	}
	_, err := b.doRequest("editMessageText", p)
	return err
}

func (b *Bot) AnswerCallback(cbID, text string) {
	p := url.Values{}
	p.Set("callback_query_id", cbID)
	if text != "" {
		p.Set("text", text)
	}
	b.doRequest("answerCallbackQuery", p)
}

func (b *Bot) DeleteMessage(chatID int64, msgID int) {
	p := url.Values{}
	p.Set("chat_id", strconv.FormatInt(chatID, 10))
	p.Set("message_id", strconv.Itoa(msgID))
	b.doRequest("deleteMessage", p)
}

func (b *Bot) CheckMember(channel string, userID int64) bool {
	p := url.Values{}
	p.Set("chat_id", channel)
	p.Set("user_id", strconv.FormatInt(userID, 10))
	data, err := b.doRequest("getChatMember", p)
	if err != nil {
		return false
	}
	var resp ChatMemberResponse
	if err = json.Unmarshal(data, &resp); err != nil || !resp.OK {
		return false
	}
	s := resp.Result.Status
	return s == "member" || s == "administrator" || s == "creator"
}

func (b *Bot) setCommands() {
	type Cmd struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	cmds := []Cmd{
		{"/start", "🚀 Botni ishga tushirish"},
		{"/menu", "🏠 Bosh menyu"},
		{"/stats", "📊 Statistikam"},
		{"/cancel", "❌ Amalni bekor qilish"},
		{"/help", "❓ Yordam"},
	}
	mj, _ := json.Marshal(cmds)
	p := url.Values{}
	p.Set("commands", string(mj))
	b.doRequest("setMyCommands", p)
}

// ============================================================
//  KLAVIATURALAR
// ============================================================

func kbInline(rows ...[]InlineKeyboardButton) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func btn(text, data string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, CallbackData: data}
}

func btnURL(text, u string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, URL: u}
}

func MainMenuKb(userID ...int64) *InlineKeyboardMarkup {
	rows := [][]InlineKeyboardButton{
		{btn("📝 Eslatmalar", "menu_notes"), btn("📖 Lug'at", "menu_vocab")},
		{btn("🧠 Quiz", "menu_quiz"), btn("⏱ Pomodoro", "menu_pomodoro")},
		{btn("🎯 Kunlik maqsad", "menu_goals"), btn("📊 Statistika", "menu_stats")},
		{btn("🏆 Reyting", "menu_leaderboard"), btn("💬 Fikr bildirish", "menu_feedback")},
		{btn("❓ Yordam", "menu_help")},
	}
	if len(userID) > 0 && isAdmin(userID[0]) {
		rows = append(rows, []InlineKeyboardButton{btn("👨‍💼 Admin panel", "menu_admin")})
	}
	return kbInline(rows...)
}

func BackKb() *InlineKeyboardMarkup {
	return kbInline([]InlineKeyboardButton{btn("🏠 Bosh menyu", "menu_main")})
}

func NotesKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("➕ Yangi eslatma", "note_add"), btn("📋 Ro'yxat", "note_list")},
		[]InlineKeyboardButton{btn("✅ Bajarildi", "note_done"), btn("🗑 O'chirish", "note_delete")},
		[]InlineKeyboardButton{btn("📊 Statistika", "note_stats"), btn("🏠 Bosh menyu", "menu_main")},
	)
}

func VocabKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("➕ So'z qo'shish", "vocab_add"), btn("📋 Ro'yxat", "vocab_list")},
		[]InlineKeyboardButton{btn("🔄 Takrorlash", "vocab_practice"), btn("📊 Statistika", "vocab_stats")},
		[]InlineKeyboardButton{btn("✅ O'rganilganlar", "vocab_learned"), btn("🏠 Bosh menyu", "menu_main")},
	)
}

func QuizCatKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("🔢 Matematika", "quiz_cat_matematika"), btn("📜 Tarix", "quiz_cat_tarix")},
		[]InlineKeyboardButton{btn("🇬🇧 Ingliz tili", "quiz_cat_ingliz"), btn("⚛ Fizika", "quiz_cat_fizika")},
		[]InlineKeyboardButton{btn("🌍 Umumiy bilim", "quiz_cat_umumiy"), btn("🎲 Aralash", "quiz_cat_all")},
		[]InlineKeyboardButton{btn("🏠 Bosh menyu", "menu_main")},
	)
}

func QuizAnswerKb(options []string, qIdx int) *InlineKeyboardMarkup {
	labels := []string{"A", "B", "C", "D"}
	rows := make([][]InlineKeyboardButton, 0, len(options)+1)
	for i, opt := range options {
		rows = append(rows, []InlineKeyboardButton{
			{Text: fmt.Sprintf("%s) %s", labels[i], opt), CallbackData: fmt.Sprintf("quiz_ans_%d_%d", qIdx, i)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{btn("❌ Testni to'xtatish", "quiz_stop")})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func PomodoroKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("▶️ 25 daqiqa", "pomo_25"), btn("▶️ 50 daqiqa", "pomo_50")},
		[]InlineKeyboardButton{btn("⚙️ O'rnatish", "pomo_custom"), btn("📊 Hisobot", "pomo_report")},
		[]InlineKeyboardButton{btn("🏠 Bosh menyu", "menu_main")},
	)
}

func GoalsKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("➕ Maqsad qo'shish", "goal_add"), btn("📋 Maqsadlar", "goal_list")},
		[]InlineKeyboardButton{btn("✅ Bajarildi", "goal_done"), btn("📊 Natijalar", "goal_stats")},
		[]InlineKeyboardButton{btn("🏠 Bosh menyu", "menu_main")},
	)
}

func YesNoKb(yes, no string) *InlineKeyboardMarkup {
	return kbInline([]InlineKeyboardButton{btn("✅ Ha", yes), btn("❌ Yo'q", no)})
}

func AdminKb() *InlineKeyboardMarkup {
	return kbInline(
		[]InlineKeyboardButton{btn("👥 Foydalanuvchilar", "adm_users"), btn("💬 Fikrlar", "adm_feedbacks")},
		[]InlineKeyboardButton{btn("📊 Bot statistikasi", "adm_stats"), btn("📢 Xabar yuborish", "adm_broadcast")},
		[]InlineKeyboardButton{btn("🆕 So'nggi fikrlar", "adm_recent_fb"), btn("🏠 Bosh menyu", "menu_main")},
	)
}

func PhoneKb() *ReplyKeyboardMarkup {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{{Text: "📱 Raqamimni ulashish", RequestContact: true}},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}
}

func ChannelKb(channels []ChannelInfo) *InlineKeyboardMarkup {
	rows := make([][]InlineKeyboardButton, 0, len(channels)+1)
	for _, ch := range channels {
		link := "https://t.me/" + strings.TrimPrefix(ch.Username, "@")
		rows = append(rows, []InlineKeyboardButton{btnURL("📢 "+ch.Title+" — "+ch.Username, link)})
	}
	rows = append(rows, []InlineKeyboardButton{btn("✅ Tekshirish", "verify_check")})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

func RemoveKb() *ReplyKeyboardRemove {
	return &ReplyKeyboardRemove{RemoveKeyboard: true}
}

// ============================================================
//  YORDAMCHI FUNKSIYALAR
// ============================================================

func isAdmin(userID int64) bool {
	for _, id := range AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func fmtDur(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%d daqiqa %d soniya", m, s)
	}
	return fmt.Sprintf("%d soniya", s)
}

func progressBar(done, total int) string {
	if total <= 0 {
		return ""
	}
	filled := done
	if filled < 0 {
		filled = 0
	}
	if filled > total {
		filled = total
	}
	return strings.Repeat("🟩", filled) + strings.Repeat("⬜", total-filled)
}

func fullName(fn, ln string) string {
	return strings.TrimSpace(fn + " " + ln)
}

func getBotToken() string {
	if t := os.Getenv("TELEGRAM_BOT_TOKEN"); t != "" {
		return t
	}
	return BotToken
}

// ============================================================
//  TASDIQLASH TIZIMI
// ============================================================

func (b *Bot) startVerification(sess *UserSession) {
	sess.State = StateWaitingPhone
	b.storage.mu.Lock()
	b.storage.setRaw(sess)
	b.storage.mu.Unlock()

	text := fmt.Sprintf(`🔐 <b>Ro'yxatdan o'tish</b>

Salom, <b>%s</b>! 👋

Botdan foydalanish uchun 2 bosqichli tasdiqlash kerak:

📱 <b>1-bosqich:</b> Telefon raqamingizni ulashing
📢 <b>2-bosqich:</b> 3 ta kanalga a'zo bo'ling

Quyidagi tugmani bosing:`, sess.FirstName)
	b.SendMessage(sess.ChatID, text, PhoneKb())
}

func (b *Bot) handlePhoneContact(sess *UserSession, msg *Message) {
	if msg.Contact == nil {
		b.SendMessage(sess.ChatID, "❌ Iltimos, tugmani bosib raqamingizni ulashing!", PhoneKb())
		return
	}
	// Faqat o'z raqamini ulashishi kerak
	if msg.Contact.UserID != 0 && msg.Contact.UserID != sess.UserID {
		b.SendMessage(sess.ChatID, "❌ Faqat o'z raqamingizni ulashing!", PhoneKb())
		return
	}
	phone := msg.Contact.PhoneNumber
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	sess.Phone = phone
	sess.TempData["phone"] = phone
	sess.State = StateVerifyChannels

	b.storage.mu.Lock()
	b.storage.setRaw(sess)
	b.storage.mu.Unlock()

	// Telefon xabarini o'chirish
	b.DeleteMessage(sess.ChatID, msg.MessageID)
	b.showChannelStep(sess)
}

func (b *Bot) showChannelStep(sess *UserSession) {
	var sb strings.Builder
	sb.WriteString("📢 <b>Kanallarga a'zo bo'ling:</b>\n\n")
	for i, ch := range RequiredChannels {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, ch.Title))
	}
	sb.WriteString("\nA'zo bo'lgach <b>✅ Tekshirish</b> tugmasini bosing:")
	b.SendMessage(sess.ChatID, sb.String(), ChannelKb(RequiredChannels))
}

func (b *Bot) handleVerifyCheck(sess *UserSession, msgID int) {
	var notJoined []ChannelInfo
	for _, ch := range RequiredChannels {
		if !b.CheckMember(ch.Username, sess.UserID) {
			notJoined = append(notJoined, ch)
		}
	}
	if len(notJoined) > 0 {
		var sb strings.Builder
		sb.WriteString("❌ <b>Quyidagi kanallarga a'zo emassiz:</b>\n\n")
		for _, ch := range notJoined {
			sb.WriteString(fmt.Sprintf("• %s (%s)\n", ch.Title, ch.Username))
		}
		sb.WriteString("\nA'zo bo'lib ✅ Tekshirish tugmasini bosing!")
		b.EditMessage(sess.ChatID, msgID, sb.String(), ChannelKb(RequiredChannels))
		return
	}

	// Tasdiqlash muvaffaqiyatli
	sess.Verified = true
	sess.State = StateNone
	phone := sess.Phone

	b.storage.mu.Lock()
	b.storage.setRaw(sess)
	b.storage.mu.Unlock()

	go b.db.SetVerified(sess.UserID, phone)
	go b.db.UpsertUser(sess.UserID, sess.ChatID, sess.FirstName, sess.LastName, sess.Username, phone, true)

	// Kanal xabarini o'chirish
	b.DeleteMessage(sess.ChatID, msgID)

	b.SendMessage(sess.ChatID, "✅ <b>Tasdiqlash muvaffaqiyatli!</b>\n\nEndi botdan to'liq foydalanishingiz mumkin! 🎉", RemoveKb())
	b.SendMessage(sess.ChatID,
		fmt.Sprintf("🎓 <b>%s ga xush kelibsiz!</b>\n\nMenyudan kerakli bo'limni tanlang:", AppName),
		MainMenuKb(sess.UserID))
}

// ============================================================
//  HANDLER TIZIMI
// ============================================================

func (b *Bot) HandleUpdate(upd Update) {
	if upd.Message != nil {
		b.HandleMessage(upd.Message)
	} else if upd.CallbackQuery != nil {
		b.HandleCallback(upd.CallbackQuery)
	}
}

func (b *Bot) HandleMessage(msg *Message) {
	if msg.From == nil || msg.Chat == nil {
		return
	}
	sess := b.storage.GetOrCreate(msg.From, msg.Chat.ID)

	// DB ga foydalanuvchi yozish (asinxron)
	go b.db.UpsertUser(sess.UserID, sess.ChatID, sess.FirstName, sess.LastName, sess.Username, sess.Phone, sess.Verified)

	// Agar contact kelsa
	if msg.Contact != nil {
		if sess.State == StateWaitingPhone {
			b.handlePhoneContact(sess, msg)
		}
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}
	log.Printf("[MSG] user=%d state=%s text=%q verified=%v", sess.UserID, sess.State, text, sess.Verified)

	// Har doim ishlaydigan komandalar
	switch text {
	case "/start":
		b.cmdStart(sess, msg)
		return
	case "/cancel":
		sess.State = StateNone
		sess.TempData = make(map[string]string)
		b.storage.SetSession(sess)
		b.SendMessage(sess.ChatID, "❌ Amal bekor qilindi.", BackKb())
		return
	case "/admin":
		b.cmdAdmin(sess)
		return
	}

	// Tasdiqlangan bo'lmasa
	if !sess.Verified {
		if sess.State == StateWaitingPhone {
			b.SendMessage(sess.ChatID, "📱 Iltimos, tugmani bosib raqamingizni ulashing!", PhoneKb())
			return
		}
		if sess.State == StateVerifyChannels {
			b.SendMessage(sess.ChatID, "📢 Kanallarga a'zo bo'lib ✅ Tekshirish tugmasini bosing!", ChannelKb(RequiredChannels))
			return
		}
		b.startVerification(sess)
		return
	}

	// Tasdiqlangan foydalanuvchilar uchun komandalar
	switch text {
	case "/menu":
		b.SendMessage(sess.ChatID, "🏠 <b>Bosh menyu</b>", MainMenuKb(sess.UserID))
		return
	case "/help":
		b.SendMessage(sess.ChatID, buildHelp(), BackKb())
		return
	case "/stats":
		b.SendMessage(sess.ChatID, buildStatsText(sess), BackKb())
		return
	}

	// Holat asosida
	switch sess.State {
	case StateWaitingNote:
		b.noteInput(sess, text)
	case StateWaitingWordUz:
		b.vocabUzInput(sess, text)
	case StateWaitingWordEn:
		b.vocabEnInput(sess, text)
	case StateWaitingWordDesc:
		b.vocabDescInput(sess, text)
	case StateWaitingPomodoro:
		b.pomodoroInput(sess, text)
	case StateWaitingGoal:
		b.goalInput(sess, text)
	case StateWaitingGoalMinute:
		b.goalMinInput(sess, text)
	case StateWaitingFeedback:
		b.feedbackInput(sess, text, msg)
	case StateWaitingDeleteNote:
		b.deleteNoteInput(sess, text)
	case StateAdminBroadcast:
		b.adminBroadcastInput(sess, text)
	default:
		b.SendMessage(sess.ChatID, "❓ Noma'lum buyruq. /menu yozing.", BackKb())
	}
}

func (b *Bot) HandleCallback(cb *CallbackQuery) {
	if cb.From == nil || cb.Message == nil {
		return
	}
	sess := b.storage.GetOrCreate(cb.From, cb.Message.Chat.ID)
	b.AnswerCallback(cb.ID, "")
	data := cb.Data
	log.Printf("[CB] user=%d data=%q verified=%v", sess.UserID, data, sess.Verified)

	// Verify check
	if data == "verify_check" {
		if sess.State == StateVerifyChannels || !sess.Verified {
			b.handleVerifyCheck(sess, cb.Message.MessageID)
		}
		return
	}

	// Admin
	if strings.HasPrefix(data, "adm_") {
		b.adminCallback(sess, data, cb.Message.MessageID)
		return
	}

	// Tasdiqlangan emas
	if !sess.Verified {
		b.AnswerCallback(cb.ID, "❌ Avval ro'yxatdan o'ting!")
		b.startVerification(sess)
		return
	}

	// Quiz javob
	if strings.HasPrefix(data, "quiz_ans_") {
		b.quizAnswer(sess, data, cb.Message.MessageID)
		return
	}
	// Quiz keyingisi
	if strings.HasPrefix(data, "quiz_next_") {
		b.quizNext(sess, cb.Message.MessageID)
		return
	}
	if data == "quiz_finish" {
		b.quizFinish(sess, cb.Message.MessageID)
		return
	}

	switch data {
	case "menu_main":
		b.EditMessage(sess.ChatID, cb.Message.MessageID, "🏠 <b>Bosh menyu</b>", MainMenuKb(sess.UserID))
	case "menu_notes":
		b.showNotesMenu(sess, cb.Message.MessageID)
	case "menu_vocab":
		b.showVocabMenu(sess, cb.Message.MessageID)
	case "menu_quiz":
		b.showQuizMenu(sess, cb.Message.MessageID)
	case "menu_pomodoro":
		b.showPomodoroMenu(sess, cb.Message.MessageID)
	case "menu_goals":
		b.showGoalsMenu(sess, cb.Message.MessageID)
	case "menu_stats":
		b.EditMessage(sess.ChatID, cb.Message.MessageID, buildStatsText(sess), BackKb())
	case "menu_leaderboard":
		b.showLeaderboard(sess, cb.Message.MessageID)
	case "menu_feedback":
		b.startFeedback(sess, cb.Message.MessageID)
	case "menu_admin":
		if isAdmin(sess.UserID) {
			b.cmdAdmin(sess)
		}
	case "menu_help":
		b.EditMessage(sess.ChatID, cb.Message.MessageID, buildHelp(), BackKb())
	// NOTES
	case "note_add":
		b.startAddNote(sess, cb.Message.MessageID)
	case "note_list":
		b.showNoteList(sess, cb.Message.MessageID)
	case "note_done":
		b.showNoteDoneList(sess, cb.Message.MessageID)
	case "note_delete":
		b.startDeleteNote(sess, cb.Message.MessageID)
	case "note_stats":
		b.showNoteStats(sess, cb.Message.MessageID)
	// VOCAB
	case "vocab_add":
		b.startAddVocab(sess, cb.Message.MessageID)
	case "vocab_list":
		b.showVocabList(sess, cb.Message.MessageID)
	case "vocab_practice":
		b.startVocabPractice(sess, cb.Message.MessageID)
	case "vocab_stats":
		b.showVocabStats(sess, cb.Message.MessageID)
	case "vocab_learned":
		b.showLearnedVocab(sess, cb.Message.MessageID)
	// QUIZ
	case "quiz_stop":
		b.EditMessage(sess.ChatID, cb.Message.MessageID, "❓ <b>Testni to'xtatmoqchimisiz?</b>", YesNoKb("quiz_stop_yes", "quiz_stop_no"))
	case "quiz_stop_yes":
		b.quizStop(sess, cb.Message.MessageID)
	case "quiz_stop_no":
		b.sendQuizQ(sess, cb.Message.MessageID)
	case "quiz_restart":
		b.showQuizMenu(sess, cb.Message.MessageID)
	// POMODORO
	case "pomo_25":
		b.startPomodoro(sess, 25, cb.Message.MessageID)
	case "pomo_50":
		b.startPomodoro(sess, 50, cb.Message.MessageID)
	case "pomo_custom":
		b.startCustomPomo(sess, cb.Message.MessageID)
	case "pomo_report":
		b.showPomoReport(sess, cb.Message.MessageID)
	case "pomo_stop":
		b.stopPomodoro(sess, cb.Message.MessageID)
	case "pomo_check":
		b.checkPomodoro(sess, cb.Message.MessageID)
	// GOALS
	case "goal_add":
		b.startAddGoal(sess, cb.Message.MessageID)
	case "goal_list":
		b.showGoalList(sess, cb.Message.MessageID)
	case "goal_done":
		b.showGoalDoneList(sess, cb.Message.MessageID)
	case "goal_stats":
		b.showGoalStats(sess, cb.Message.MessageID)
	default:
		// prefix asosida
		switch {
		case strings.HasPrefix(data, "note_mark_"):
			b.markNoteDone(sess, data, cb.Message.MessageID)
		case strings.HasPrefix(data, "vocab_mark_"):
			b.markVocabLearned(sess, data, cb.Message.MessageID)
		case strings.HasPrefix(data, "vocab_next_"):
			b.startVocabPractice(sess, cb.Message.MessageID)
		case strings.HasPrefix(data, "quiz_cat_"):
			b.startQuiz(sess, data, cb.Message.MessageID)
		case strings.HasPrefix(data, "goal_mark_"):
			b.markGoalDone(sess, data, cb.Message.MessageID)
		default:
			log.Printf("[CB] Noma'lum: %q", data)
		}
	}
}

// ============================================================
//  /START
// ============================================================

func (b *Bot) cmdStart(sess *UserSession, msg *Message) {
	if !sess.Verified {
		b.startVerification(sess)
		return
	}
	text := fmt.Sprintf(`🎓 <b>%s ga xush kelibsiz!</b>

Salom, <b>%s</b>! 👋

Menyudan kerakli bo'limni tanlang 🚀`, AppName, msg.From.FirstName)
	b.SendMessage(sess.ChatID, text, MainMenuKb(sess.UserID))
}

func (b *Bot) cmdAdmin(sess *UserSession) {
	if !isAdmin(sess.UserID) {
		b.SendMessage(sess.ChatID, "❌ Ruxsat yo'q.", nil)
		return
	}
	total := b.db.CountUsers()
	ver := b.db.CountVerified()
	fb := b.db.CountFeedbacks()
	text := fmt.Sprintf(`👨‍💼 <b>Admin Panel</b>

📊 <b>Umumiy:</b>
👥 Jami foydalanuvchilar: <b>%d</b>
✅ Tasdiqlangan: <b>%d</b>
❌ Tasdiqlanmagan: <b>%d</b>
💬 Jami fikrlar: <b>%d</b>

🕐 %s`, total, ver, total-ver, fb, time.Now().Format("02.01.2006 15:04"))
	b.SendMessage(sess.ChatID, text, AdminKb())
}

// ============================================================
//  ESLATMALAR
// ============================================================

func (b *Bot) showNotesMenu(sess *UserSession, msgID int) {
	done := 0
	for _, n := range sess.Notes {
		if n.Done {
			done++
		}
	}
	total := len(sess.Notes)
	text := fmt.Sprintf("📝 <b>Eslatmalar</b>\n\n📊 Jami: <b>%d</b>\n✅ Bajarilgan: <b>%d</b>\n⏳ Kutmoqda: <b>%d</b>\n\nAmalni tanlang:", total, done, total-done)
	b.EditMessage(sess.ChatID, msgID, text, NotesKb())
}

func (b *Bot) startAddNote(sess *UserSession, msgID int) {
	sess.State = StateWaitingNote
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "📝 <b>Yangi eslatma</b>\n\nMatnini kiriting:\n<i>(/cancel bekor qilish)</i>", BackKb())
}

func (b *Bot) noteInput(sess *UserSession, text string) {
	if len([]rune(text)) < 2 {
		b.SendMessage(sess.ChatID, "❌ Juda qisqa! Kamida 2 belgi.", nil)
		return
	}
	if len([]rune(text)) > 500 {
		b.SendMessage(sess.ChatID, "❌ 500 belgidan oshmasin.", nil)
		return
	}
	id := len(sess.Notes) + 1
	sess.Notes = append(sess.Notes, Note{ID: id, Text: text, CreatedAt: time.Now(), Priority: "medium"})
	sess.Stats.NotesCreated++
	sess.State = StateNone
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID, fmt.Sprintf("✅ <b>Eslatma qo'shildi!</b>\n\n📝 #%d: %s\n🕐 %s", id, text, time.Now().Format("02.01.2006 15:04")), NotesKb())
}

func (b *Bot) showNoteList(sess *UserSession, msgID int) {
	if len(sess.Notes) == 0 {
		b.EditMessage(sess.ChatID, msgID, "📋 <b>Eslatmalar yo'q.</b>", NotesKb())
		return
	}
	var sb strings.Builder
	sb.WriteString("📋 <b>Barcha eslatmalar:</b>\n\n")
	for i, n := range sess.Notes {
		if i >= 20 {
			sb.WriteString(fmt.Sprintf("... va yana %d ta", len(sess.Notes)-20))
			break
		}
		s := "⏳"
		if n.Done {
			s = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s #%d: %s\n   📅 %s\n\n", s, n.ID, n.Text, n.CreatedAt.Format("02.01 15:04")))
	}
	b.EditMessage(sess.ChatID, msgID, sb.String(), NotesKb())
}

func (b *Bot) showNoteDoneList(sess *UserSession, msgID int) {
	var pending []Note
	for _, n := range sess.Notes {
		if !n.Done {
			pending = append(pending, n)
		}
	}
	if len(pending) == 0 {
		b.EditMessage(sess.ChatID, msgID, "✅ <b>Bajarilmagan yo'q!</b>\n\nBarchasi tugallangan. 🎉", NotesKb())
		return
	}
	rows := make([][]InlineKeyboardButton, 0, len(pending)+1)
	for _, n := range pending {
		rows = append(rows, []InlineKeyboardButton{
			{Text: fmt.Sprintf("✅ #%d: %s", n.ID, trunc(n.Text, 30)), CallbackData: fmt.Sprintf("note_mark_%d", n.ID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{btn("🔙 Orqaga", "menu_notes")})
	b.EditMessage(sess.ChatID, msgID, "⏳ <b>Bajarilmaganlar:</b>", &InlineKeyboardMarkup{InlineKeyboard: rows})
}

func (b *Bot) markNoteDone(sess *UserSession, data string, msgID int) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	for i, n := range sess.Notes {
		if n.ID == id {
			sess.Notes[i].Done = true
			sess.Stats.NotesDone++
			b.storage.SetSession(sess)
			b.EditMessage(sess.ChatID, msgID, fmt.Sprintf("✅ <b>Bajarildi!</b>\n\n📝 %s\n\nAjoyib! 🎉", n.Text), NotesKb())
			return
		}
	}
}

func (b *Bot) startDeleteNote(sess *UserSession, msgID int) {
	if len(sess.Notes) == 0 {
		b.EditMessage(sess.ChatID, msgID, "📋 O'chirish uchun eslatma yo'q.", NotesKb())
		return
	}
	sess.State = StateWaitingDeleteNote
	b.storage.SetSession(sess)
	var sb strings.Builder
	sb.WriteString("🗑 <b>O'chirish — raqam kiriting:</b>\n\n")
	for _, n := range sess.Notes {
		s := "⏳"
		if n.Done {
			s = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s #%d: %s\n", s, n.ID, trunc(n.Text, 35)))
	}
	sb.WriteString("\n<i>(/cancel bekor qilish)</i>")
	b.EditMessage(sess.ChatID, msgID, sb.String(), nil)
}

func (b *Bot) deleteNoteInput(sess *UserSession, text string) {
	id, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		b.SendMessage(sess.ChatID, "❌ Faqat raqam kiriting!", nil)
		return
	}
	for i, n := range sess.Notes {
		if n.ID == id {
			sess.Notes = append(sess.Notes[:i], sess.Notes[i+1:]...)
			sess.State = StateNone
			if sess.LastMessageID != 0 {
				b.DeleteMessage(sess.ChatID, sess.LastMessageID)
			}
			b.storage.SetSession(sess)
			b.SendMessage(sess.ChatID, fmt.Sprintf("🗑 <b>#%d eslatma o'chirildi.</b>", id), NotesKb())
			return
		}
	}
	b.SendMessage(sess.ChatID, fmt.Sprintf("❌ #%d topilmadi.", id), nil)
}

func (b *Bot) showNoteStats(sess *UserSession, msgID int) {
	total := len(sess.Notes)
	done := 0
	for _, n := range sess.Notes {
		if n.Done {
			done++
		}
	}
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	text := fmt.Sprintf("📊 <b>Eslatmalar statistikasi</b>\n\n📋 Jami: <b>%d</b>\n✅ Bajarilgan: <b>%d</b>\n⏳ Qolgan: <b>%d</b>\n📈 Foiz: <b>%d%%</b>\n\n🏆 Jami yaratilgan: <b>%d</b>\n🎯 Jami bajarilgan: <b>%d</b>",
		total, done, total-done, pct, sess.Stats.NotesCreated, sess.Stats.NotesDone)
	b.EditMessage(sess.ChatID, msgID, text, NotesKb())
}

// ============================================================
//  LUG'AT
// ============================================================

func (b *Bot) showVocabMenu(sess *UserSession, msgID int) {
	learned := 0
	for _, w := range sess.Vocabulary {
		if w.Learned {
			learned++
		}
	}
	total := len(sess.Vocabulary)
	b.EditMessage(sess.ChatID, msgID,
		fmt.Sprintf("📖 <b>Lug'at</b>\n\n📚 Jami: <b>%d</b>\n✅ O'rganilgan: <b>%d</b>\n🔄 Qolgan: <b>%d</b>\n\nAmalni tanlang:", total, learned, total-learned),
		VocabKb())
}

func (b *Bot) startAddVocab(sess *UserSession, msgID int) {
	sess.State = StateWaitingWordUz
	sess.TempData = make(map[string]string)
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "📖 <b>Yangi so'z</b>\n\n1️⃣ O'zbekcha so'zni kiriting:\n<i>(/cancel bekor qilish)</i>", nil)
}

func (b *Bot) vocabUzInput(sess *UserSession, text string) {
	if strings.TrimSpace(text) == "" {
		b.SendMessage(sess.ChatID, "❌ Bo'sh bo'lmasin!", nil)
		return
	}
	sess.TempData["uz"] = text
	sess.State = StateWaitingWordEn
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID, fmt.Sprintf("O'zbekcha: <b>%s</b>\n\n2️⃣ Inglizcha tarjimasini kiriting:", text), nil)
}

func (b *Bot) vocabEnInput(sess *UserSession, text string) {
	if strings.TrimSpace(text) == "" {
		b.SendMessage(sess.ChatID, "❌ Bo'sh bo'lmasin!", nil)
		return
	}
	sess.TempData["en"] = text
	sess.State = StateWaitingWordDesc
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID, fmt.Sprintf("🇺🇿 <b>%s</b> — 🇬🇧 <b>%s</b>\n\n3️⃣ Izoh (ixtiyoriy, o'tkazish uchun \"-\"):", sess.TempData["uz"], text), nil)
}

func (b *Bot) vocabDescInput(sess *UserSession, text string) {
	desc := ""
	if text != "-" {
		desc = text
	}
	id := len(sess.Vocabulary) + 1
	w := VocabWord{ID: id, Uzbek: sess.TempData["uz"], English: sess.TempData["en"], Desc: desc, AddedAt: time.Now()}
	sess.Vocabulary = append(sess.Vocabulary, w)
	sess.Stats.WordsAdded++
	sess.State = StateNone
	sess.TempData = make(map[string]string)
	b.storage.SetSession(sess)
	msg := fmt.Sprintf("✅ <b>So'z qo'shildi!</b>\n\n🇺🇿 <b>%s</b>\n🇬🇧 <b>%s</b>", w.Uzbek, w.English)
	if w.Desc != "" {
		msg += "\n📝 <i>" + w.Desc + "</i>"
	}
	b.SendMessage(sess.ChatID, msg, VocabKb())
}

func (b *Bot) showVocabList(sess *UserSession, msgID int) {
	if len(sess.Vocabulary) == 0 {
		b.EditMessage(sess.ChatID, msgID, "📋 <b>Lug'at bo'sh.</b>", VocabKb())
		return
	}
	var sb strings.Builder
	sb.WriteString("📖 <b>So'zlar ro'yxati:</b>\n\n")
	for i, w := range sess.Vocabulary {
		if i >= 15 {
			sb.WriteString(fmt.Sprintf("... va yana %d ta", len(sess.Vocabulary)-15))
			break
		}
		s := "🔄"
		if w.Learned {
			s = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %s\n", s, w.Uzbek, trunc(w.English, 30)))
		if w.Desc != "" {
			sb.WriteString(fmt.Sprintf("   <i>%s</i>\n", trunc(w.Desc, 50)))
		}
	}
	b.EditMessage(sess.ChatID, msgID, sb.String(), VocabKb())
}

func (b *Bot) startVocabPractice(sess *UserSession, msgID int) {
	var unlearned []VocabWord
	for _, w := range sess.Vocabulary {
		if !w.Learned {
			unlearned = append(unlearned, w)
		}
	}
	if len(unlearned) == 0 {
		b.EditMessage(sess.ChatID, msgID, "🎉 <b>Barcha so'zlarni o'rgandingiz!</b>\n\nYangi so'zlar qo'shing.", VocabKb())
		return
	}
	rand.Shuffle(len(unlearned), func(i, j int) { unlearned[i], unlearned[j] = unlearned[j], unlearned[i] })
	w := unlearned[0]
	sess.TempData["pw"] = strconv.Itoa(w.ID)
	b.storage.SetSession(sess)
	text := fmt.Sprintf("🔄 <b>Takrorlash</b>\n\n❓ Inglizchasi nima?\n\n🇺🇿 <b>%s</b>", w.Uzbek)
	kb := kbInline(
		[]InlineKeyboardButton{{Text: "✅ Ha: " + w.English, CallbackData: fmt.Sprintf("vocab_mark_%d", w.ID)}},
		[]InlineKeyboardButton{btn("➡️ Keyingisi", fmt.Sprintf("vocab_next_%d", w.ID)), btn("🔙 Orqaga", "menu_vocab")},
	)
	b.EditMessage(sess.ChatID, msgID, text, kb)
}

func (b *Bot) markVocabLearned(sess *UserSession, data string, msgID int) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	for i, w := range sess.Vocabulary {
		if w.ID == id {
			sess.Vocabulary[i].Learned = true
			sess.Vocabulary[i].LearnCount++
			sess.Stats.WordsLearned++
			b.storage.SetSession(sess)
			b.EditMessage(sess.ChatID, msgID, fmt.Sprintf("✅ <b>Zo'r!</b>\n\n🇺🇿 <b>%s</b> — 🇬🇧 <b>%s</b>\n\nO'rganildi! 🎉", w.Uzbek, w.English), VocabKb())
			return
		}
	}
}

func (b *Bot) showVocabStats(sess *UserSession, msgID int) {
	total := len(sess.Vocabulary)
	learned := 0
	for _, w := range sess.Vocabulary {
		if w.Learned {
			learned++
		}
	}
	pct := 0
	if total > 0 {
		pct = learned * 100 / total
	}
	b.EditMessage(sess.ChatID, msgID,
		fmt.Sprintf("📊 <b>Lug'at statistikasi</b>\n\n📚 Jami: <b>%d</b>\n✅ O'rganilgan: <b>%d</b>\n🔄 Qolgan: <b>%d</b>\n📈 Foiz: <b>%d%%</b>", total, learned, total-learned, pct),
		VocabKb())
}

func (b *Bot) showLearnedVocab(sess *UserSession, msgID int) {
	var sb strings.Builder
	sb.WriteString("✅ <b>O'rganilgan so'zlar:</b>\n\n")

	var learned []VocabWord
	for _, w := range sess.Vocabulary {
		if w.Learned {
			learned = append(learned, w)
		}
	}

	if len(learned) == 0 {
		sb.WriteString("<i>Hali o'rganilmagan.</i>")
	} else {
		limit := 15
		if len(learned) < limit {
			limit = len(learned)
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("🇺🇿 <b>%s</b> — 🇬🇧 %s\n", learned[i].Uzbek, learned[i].English))
		}
		if len(learned) > 15 {
			sb.WriteString(fmt.Sprintf("\n... va yana <b>%d</b> ta", len(learned)-15))
		}
	}
	b.EditMessage(sess.ChatID, msgID, sb.String(), VocabKb())
}

// ============================================================
//  QUIZ
// ============================================================

func (b *Bot) showQuizMenu(sess *UserSession, msgID int) {
	text := fmt.Sprintf("🧠 <b>Quiz - Bilim sinovi</b>\n\n🏆 Rekord: <b>%d</b> ball\n📊 O'yinlar: <b>%d</b>\n⭐ Jami: <b>%d</b>\n\nKategoriya tanlang:",
		sess.Stats.BestScore, sess.Stats.TotalQuiz, sess.Stats.TotalScore)
	b.EditMessage(sess.ChatID, msgID, text, QuizCatKb())
}

func (b *Bot) startQuiz(sess *UserSession, data string, msgID int) {
	cat := "all"
	switch strings.TrimPrefix(data, "quiz_cat_") {
	case "matematika":
		cat = CategoryMath
	case "tarix":
		cat = CategoryHistory
	case "ingliz":
		cat = CategoryEnglish
	case "fizika":
		cat = CategoryScience
	case "umumiy":
		cat = CategoryGeneral
	}
	qs := GetQuestionsByCategory(cat)
	if len(qs) == 0 {
		b.EditMessage(sess.ChatID, msgID, "❌ Savol topilmadi.", BackKb())
		return
	}
	cnt := 5
	if cat == "all" {
		cnt = 10
	}
	sess.Quiz = &QuizSession{Questions: ShuffleQuestions(qs, cnt), Category: cat, StartTime: time.Now()}
	sess.State = StateQuizActive
	b.storage.SetSession(sess)
	b.sendQuizQ(sess, msgID)
}

func (b *Bot) sendQuizQ(sess *UserSession, msgID int) {
	if sess.Quiz == nil || sess.Quiz.CurrentIndex >= len(sess.Quiz.Questions) {
		b.quizFinish(sess, msgID)
		return
	}
	q := sess.Quiz.Questions[sess.Quiz.CurrentIndex]
	total := len(sess.Quiz.Questions)
	cur := sess.Quiz.CurrentIndex + 1
	bar := progressBar(cur-1, total)
	text := fmt.Sprintf("🧠 <b>Quiz</b> | %d/%d\n\n%s\n\n❓ <b>%s</b>", cur, total, bar, q.Question)
	b.EditMessage(sess.ChatID, msgID, text, QuizAnswerKb(q.Options, sess.Quiz.CurrentIndex))
}

func (b *Bot) quizAnswer(sess *UserSession, data string, msgID int) {
	if sess.Quiz == nil {
		b.EditMessage(sess.ChatID, msgID, "❌ Quiz topilmadi.", MainMenuKb(sess.UserID))
		return
	}
	parts := strings.Split(data, "_")
	// format: quiz_ans_<qIdx>_<ansIdx>
	if len(parts) != 4 {
		return
	}
	qIdx, e1 := strconv.Atoi(parts[2])
	ansIdx, e2 := strconv.Atoi(parts[3])
	if e1 != nil || e2 != nil {
		return
	}
	// Eski tugma bosilmasin
	if qIdx != sess.Quiz.CurrentIndex {
		return
	}
	// Savol chegarasini tekshirish
	if qIdx >= len(sess.Quiz.Questions) {
		b.quizFinish(sess, msgID)
		return
	}

	q := sess.Quiz.Questions[qIdx]
	labels := []string{"A", "B", "C", "D"}
	sess.Quiz.Answered++

	var resultText string
	if ansIdx == q.Correct {
		sess.Quiz.Score += q.Points
		sess.Quiz.Correct++
		resultText = fmt.Sprintf(
			"✅ <b>To'g'ri!</b> +%d ball\n\n💡 <i>%s</i>\n\n📊 %d/%d to'g'ri | %d ball",
			q.Points, q.Explain, sess.Quiz.Correct, sess.Quiz.Answered, sess.Quiz.Score)
	} else {
		resultText = fmt.Sprintf(
			"❌ <b>Noto'g'ri!</b>\n\n✅ To'g'ri: <b>%s) %s</b>\n💡 <i>%s</i>\n\n📊 %d/%d to'g'ri | %d ball",
			labels[q.Correct], q.Options[q.Correct], q.Explain,
			sess.Quiz.Correct, sess.Quiz.Answered, sess.Quiz.Score)
	}

	sess.Quiz.CurrentIndex++

	nextText := "➡️ Keyingisi"
	cbData := fmt.Sprintf("quiz_next_%d", sess.Quiz.CurrentIndex)
	if sess.Quiz.CurrentIndex >= len(sess.Quiz.Questions) {
		nextText = "🏁 Natija ko'rish"
		cbData = "quiz_finish"
	}
	kb := kbInline([]InlineKeyboardButton{
		{Text: nextText, CallbackData: cbData},
	})
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, resultText, kb)
}

func (b *Bot) quizNext(sess *UserSession, msgID int) {
	if sess.Quiz == nil || sess.Quiz.CurrentIndex >= len(sess.Quiz.Questions) {
		b.quizFinish(sess, msgID)
		return
	}
	b.sendQuizQ(sess, msgID)
}

func (b *Bot) quizFinish(sess *UserSession, msgID int) {
	if sess.Quiz == nil {
		return
	}
	// Barcha qiymatlarni nil dan OLDIN olamiz
	score := sess.Quiz.Score
	total := len(sess.Quiz.Questions)
	answered := sess.Quiz.Answered
	correct := sess.Quiz.Correct
	dur := time.Since(sess.Quiz.StartTime)

	// Statistikani yangilash
	if score > sess.Stats.BestScore {
		sess.Stats.BestScore = score
	}
	sess.Stats.TotalScore += score
	sess.Stats.TotalQuiz++
	sess.Quiz = nil
	sess.State = StateNone
	b.storage.SetSession(sess)

	// Reyting
	rating := "🥉"
	comment := "Davom eting! 💫"
	if correct == total {
		rating = "🏆"
		comment = "Barcha javob to'g'ri! Dahosiz! 🔥"
	} else if score >= 80 {
		rating = "🥇"
		comment = "Zo'r natija! 💪"
	} else if score >= 50 {
		rating = "🥈"
		comment = "Yaxshi harakat! 📚"
	}

	accuracy := 0
	if answered > 0 {
		accuracy = correct * 100 / answered
	}

	text := fmt.Sprintf(
		"🏁 <b>Quiz yakunlandi!</b> %s\n\n📊 Savollar: <b>%d</b>\n✅ To'g'ri: <b>%d</b>\n❌ Noto'g'ri: <b>%d</b>\n🎯 Aniqlik: <b>%d%%</b>\n⭐ Ball: <b>%d</b>\n⏱ Vaqt: <b>%s</b>\n\n💬 %s",
		rating, total, correct, answered-correct, accuracy, score, fmtDur(dur), comment)
	kb := kbInline([]InlineKeyboardButton{btn("🔄 Qayta", "menu_quiz"), btn("🏠 Menyu", "menu_main")})
	b.EditMessage(sess.ChatID, msgID, text, kb)
}

func (b *Bot) quizStop(sess *UserSession, msgID int) {
	if sess.Quiz != nil {
		s := sess.Quiz.Score
		c := sess.Quiz.Correct
		t := sess.Quiz.Answered
		sess.Stats.TotalScore += s
		sess.Stats.TotalQuiz++
		if s > sess.Stats.BestScore {
			sess.Stats.BestScore = s
		}
		b.storage.SetSession(sess)
		sess.Quiz = nil
		b.EditMessage(sess.ChatID, msgID, 
			fmt.Sprintf("⏹ <b>Quiz to'xtatildi</b>\n\n✅ To'g'ri: <b>%d</b>\n❌ Noto'g'ri: <b>%d</b>\n⭐ Ball: <b>%d</b>", c, t-c, s),
			MainMenuKb(sess.UserID))
		return
	}
	sess.Quiz = nil
	sess.State = StateNone
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "❌ <b>Quiz to'xtatildi.</b>", MainMenuKb(sess.UserID))
}

// ============================================================
//  POMODORO
// ============================================================

func (b *Bot) showPomodoroMenu(sess *UserSession, msgID int) {
	extra := ""
	if sess.Pomodoro != nil && sess.Pomodoro.IsRunning {
		rem := time.Duration(sess.Pomodoro.Duration)*time.Minute - time.Since(sess.Pomodoro.StartTime)
		if rem > 0 {
			extra = "\n\n⏳ <b>Faol sessiya:</b> " + fmtDur(rem) + " qoldi"
		} else {
			extra = "\n\n✅ <b>Sessiya tugadi!</b>"
		}
	}
	b.EditMessage(sess.ChatID, msgID,
		fmt.Sprintf("⏱ <b>Pomodoro texnikasi</b>%s\n\n🍅 Jami: <b>%d</b>\n\nVaqt tanlang:", extra, sess.Stats.TotalPomodoro),
		PomodoroKb())
}

func (b *Bot) startPomodoro(sess *UserSession, dur int, msgID int) {
	sess.Pomodoro = &PomodoroSession{Duration: dur, StartTime: time.Now(), Task: "O'qish", IsRunning: true}
	b.storage.SetSession(sess)
	end := time.Now().Add(time.Duration(dur) * time.Minute)
	text := fmt.Sprintf("✅ <b>Pomodoro boshlandi!</b>\n\n⏱ Muddat: <b>%d daqiqa</b>\n🕐 Tugash: <b>%s</b>\n\n💪 Diqqatingizni jamlang!", dur, end.Format("15:04"))
	kb := kbInline(
		[]InlineKeyboardButton{btn("🔍 Tekshirish", "pomo_check"), btn("❌ To'xtatish", "pomo_stop")},
		[]InlineKeyboardButton{btn("🏠 Bosh menyu", "menu_main")},
	)
	b.EditMessage(sess.ChatID, msgID, text, kb)
}

func (b *Bot) startCustomPomo(sess *UserSession, msgID int) {
	sess.State = StateWaitingPomodoro
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "⚙️ <b>Davomiylik kiriting</b> (daqiqa, 1-180):\n<i>(/cancel bekor qilish)</i>", nil)
}

func (b *Bot) pomodoroInput(sess *UserSession, text string) {
	dur, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || dur < 1 || dur > 180 {
		b.SendMessage(sess.ChatID, "❌ 1-180 oralig'ida raqam kiriting!", nil)
		return
	}
	sess.State = StateNone
	sess.Pomodoro = &PomodoroSession{Duration: dur, StartTime: time.Now(), Task: "Maxsus", IsRunning: true}
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID,
		fmt.Sprintf("✅ <b>%d daqiqalik Pomodoro!</b>\n🕐 Tugash: <b>%s</b>", dur, time.Now().Add(time.Duration(dur)*time.Minute).Format("15:04")),
		PomodoroKb())
}

func (b *Bot) checkPomodoro(sess *UserSession, msgID int) {
	if sess.Pomodoro == nil || !sess.Pomodoro.IsRunning {
		b.EditMessage(sess.ChatID, msgID, "❌ Faol sessiya yo'q.", PomodoroKb())
		return
	}
	elapsed := time.Since(sess.Pomodoro.StartTime)
	total := time.Duration(sess.Pomodoro.Duration) * time.Minute
	rem := total - elapsed
	if rem <= 0 {
		sess.Pomodoro.IsRunning = false
		sess.Stats.TotalPomodoro++
		go b.db.LogPomodoro(sess.UserID, sess.Pomodoro.Duration, true)
		b.storage.SetSession(sess)
		b.EditMessage(sess.ChatID, msgID, fmt.Sprintf("🎉 <b>Pomodoro tugadi!</b>\n\n✅ <b>%d daqiqa</b> yakunlandi!\n☕ Dam oling!", sess.Pomodoro.Duration), PomodoroKb())
		return
	}
	pct := int(elapsed * 100 / total)
	bar := progressBar(pct/10, 10)
	text := fmt.Sprintf("⏱ <b>Pomodoro davom etmoqda</b>\n\n⏳ O'tgan: <b>%s</b>\n⏰ Qolgan: <b>%s</b>\n%s %d%%",
		fmtDur(elapsed), fmtDur(rem), bar, pct)
	kb := kbInline([]InlineKeyboardButton{btn("🔍 Yangilash", "pomo_check"), btn("❌ To'xtatish", "pomo_stop")})
	b.EditMessage(sess.ChatID, msgID, text, kb)
}

func (b *Bot) stopPomodoro(sess *UserSession, msgID int) {
	if sess.Pomodoro == nil {
		b.EditMessage(sess.ChatID, msgID, "Faol sessiya yo'q.", PomodoroKb())
		return
	}
	elapsed := time.Since(sess.Pomodoro.StartTime)
	sess.Pomodoro.IsRunning = false
	sess.Stats.TotalPomodoro++
	go b.db.LogPomodoro(sess.UserID, sess.Pomodoro.Duration, false)
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, fmt.Sprintf("⏹ <b>To'xtatildi</b>\n\n⏱ Ishlangan: <b>%s</b>", fmtDur(elapsed)), PomodoroKb())
}

func (b *Bot) showPomoReport(sess *UserSession, msgID int) {
	total := sess.Stats.TotalPomodoro * 25
	b.EditMessage(sess.ChatID, msgID,
		fmt.Sprintf("📊 <b>Pomodoro hisoboti</b>\n\n🍅 Sessiyalar: <b>%d</b>\n⏱ Taxminiy: <b>%d soat %d daqiqa</b>\n\n💡 25 daqiqa ishlang, 5 daqiqa dam oling",
			sess.Stats.TotalPomodoro, total/60, total%60),
		PomodoroKb())
}

// ============================================================
//  MAQSADLAR
// ============================================================

func (b *Bot) showGoalsMenu(sess *UserSession, msgID int) {
	today := time.Now().Format("2006-01-02")
	tg, td := 0, 0
	for _, g := range sess.Goals {
		if g.Date == today {
			tg++
			if g.Done {
				td++
			}
		}
	}
	b.EditMessage(sess.ChatID, msgID,
		fmt.Sprintf("🎯 <b>Kunlik maqsadlar</b>\n\n📅 Bugun: <b>%s</b>\n📊 Maqsadlar: <b>%d</b>\n✅ Bajarilgan: <b>%d</b>\n\nAmalni tanlang:", time.Now().Format("02.01.2006"), tg, td),
		GoalsKb())
}

func (b *Bot) startAddGoal(sess *UserSession, msgID int) {
	sess.State = StateWaitingGoal
	sess.TempData = make(map[string]string)
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "🎯 <b>Yangi maqsad</b>\n\nMaqsad tavsifini kiriting:\n<i>(/cancel bekor qilish)</i>", nil)
}

func (b *Bot) goalInput(sess *UserSession, text string) {
	if len([]rune(strings.TrimSpace(text))) < 3 {
		b.SendMessage(sess.ChatID, "❌ Juda qisqa!", nil)
		return
	}
	sess.TempData["desc"] = text
	sess.State = StateWaitingGoalMinute
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID, fmt.Sprintf("🎯 Maqsad: <b>%s</b>\n\nNecha daqiqa sarflaysiz? (1-480):", text), nil)
}

func (b *Bot) goalMinInput(sess *UserSession, text string) {
	mins, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || mins < 1 || mins > 480 {
		b.SendMessage(sess.ChatID, "❌ 1-480 oralig'ida kiriting!", nil)
		return
	}
	id := len(sess.Goals) + 1
	g := DailyGoal{ID: id, Description: sess.TempData["desc"], TargetMin: mins, Date: time.Now().Format("2006-01-02"), CreatedAt: time.Now()}
	sess.Goals = append(sess.Goals, g)
	sess.Stats.TotalGoals++
	sess.State = StateNone
	sess.TempData = make(map[string]string)
	b.storage.SetSession(sess)
	b.SendMessage(sess.ChatID, fmt.Sprintf("✅ <b>Maqsad qo'shildi!</b>\n\n🎯 %s\n⏱ Reja: <b>%d daqiqa</b>", g.Description, mins), GoalsKb())
}

func (b *Bot) showGoalList(sess *UserSession, msgID int) {
	today := time.Now().Format("2006-01-02")
	var tg []DailyGoal
	for _, g := range sess.Goals {
		if g.Date == today {
			tg = append(tg, g)
		}
	}
	if len(tg) == 0 {
		b.EditMessage(sess.ChatID, msgID, "📋 <b>Bugungi maqsadlar yo'q.</b>", GoalsKb())
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Bugungi maqsadlar (%s):</b>\n\n", time.Now().Format("02.01.2006")))
	for _, g := range tg {
		s := "⏳"
		if g.Done {
			s = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s #%d: <b>%s</b> — %d daqiqa\n", s, g.ID, g.Description, g.TargetMin))
	}
	b.EditMessage(sess.ChatID, msgID, sb.String(), GoalsKb())
}

func (b *Bot) showGoalDoneList(sess *UserSession, msgID int) {
	today := time.Now().Format("2006-01-02")
	var pending []DailyGoal
	for _, g := range sess.Goals {
		if g.Date == today && !g.Done {
			pending = append(pending, g)
		}
	}
	if len(pending) == 0 {
		b.EditMessage(sess.ChatID, msgID, "🎉 <b>Bugun barcha maqsadlar bajarildi!</b>", GoalsKb())
		return
	}
	rows := make([][]InlineKeyboardButton, 0, len(pending)+1)
	for _, g := range pending {
		rows = append(rows, []InlineKeyboardButton{
			{Text: fmt.Sprintf("✅ #%d: %s", g.ID, trunc(g.Description, 30)), CallbackData: fmt.Sprintf("goal_mark_%d", g.ID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{btn("🔙 Orqaga", "menu_goals")})
	b.EditMessage(sess.ChatID, msgID, "⏳ <b>Bajarilmaganlar:</b>", &InlineKeyboardMarkup{InlineKeyboard: rows})
}

func (b *Bot) markGoalDone(sess *UserSession, data string, msgID int) {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}
	for i, g := range sess.Goals {
		if g.ID == id {
			sess.Goals[i].Done = true
			sess.Goals[i].SpentMin = g.TargetMin
			sess.Stats.GoalsDone++
			b.storage.SetSession(sess)
			b.EditMessage(sess.ChatID, msgID, fmt.Sprintf("🎉 <b>Maqsad bajarildi!</b>\n\n✅ %s\n⏱ %d daqiqa", g.Description, g.TargetMin), GoalsKb())
			return
		}
	}
}

func (b *Bot) showGoalStats(sess *UserSession, msgID int) {
	total, done := sess.Stats.TotalGoals, sess.Stats.GoalsDone
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	today := time.Now().Format("2006-01-02")
	tt, td := 0, 0
	for _, g := range sess.Goals {
		if g.Date == today {
			tt++
			if g.Done {
				td++
			}
		}
	}
	text := fmt.Sprintf("📊 <b>Maqsadlar statistikasi</b>\n\n📅 Bugun:\n  📋 Jami: <b>%d</b>\n  ✅ Bajarilgan: <b>%d</b>\n\n📈 Umuman:\n  🎯 Jami: <b>%d</b>\n  ✅ Bajarilgan: <b>%d</b>\n  📊 Samaradorlik: <b>%d%%</b>",
		tt, td, total, done, pct)
	b.EditMessage(sess.ChatID, msgID, text, GoalsKb())
}

// ============================================================
//  STATISTIKA
// ============================================================

func buildStatsText(sess *UserSession) string {
	s := sess.Stats
	avg := 0
	if s.TotalQuiz > 0 {
		avg = s.TotalScore / s.TotalQuiz
	}
	phone := sess.Phone
	if phone == "" {
		phone = "—"
	}
	uname := s.Username
	if uname == "" {
		uname = "—"
	}
	return fmt.Sprintf(`📊 <b>Sizning statistikangiz</b>

👤 <b>%s</b> | @%s
📱 Telefon: <code>%s</code>
📅 Ro'yxat: %s
🔥 Streak: <b>%d kun</b>
📆 Faol kunlar: <b>%d</b>

🧠 Quiz: <b>%d</b> o'yin | Rekord <b>%d</b> ball
📝 Eslatmalar: <b>%d</b> yaratilgan | <b>%d</b> bajarilgan
📖 Lug'at: <b>%d</b> qo'shilgan | <b>%d</b> o'rganilgan
⏱ Pomodoro: <b>%d</b> sessiya
🎯 Maqsadlar: <b>%d</b> jami | <b>%d</b> bajarilgan
📈 O'rtacha ball: <b>%d</b>`,
		s.FirstName, uname, phone,
		s.JoinedAt.Format("02.01.2006"),
		s.Streak, s.DaysActive,
		s.TotalQuiz, s.BestScore,
		s.NotesCreated, s.NotesDone,
		s.WordsAdded, s.WordsLearned,
		s.TotalPomodoro,
		s.TotalGoals, s.GoalsDone,
		avg,
	)
}

// ============================================================
//  REYTING
// ============================================================

func (b *Bot) showLeaderboard(sess *UserSession, msgID int) {
	all, err := b.db.GetLeaderboard()
	if err != nil {
		b.EditMessage(sess.ChatID, msgID, "❌ Reyting yuklanmadi.", BackKb())
		return
	}
	sort.Slice(all, func(i, j int) bool { return all[i].TotalScore > all[j].TotalScore })
	var sb strings.Builder
	sb.WriteString("🏆 <b>Reyting (Quiz bo'yicha)</b>\n\n")
	medals := []string{"🥇", "🥈", "🥉"}
	userRank := -1
	for i, st := range all {
		if st.UserID == sess.UserID {
			userRank = i + 1
		}
		if i >= 10 {
			continue
		}
		medal := fmt.Sprintf("%d.", i+1)
		if i < len(medals) {
			medal = medals[i]
		}
		name := st.FirstName
		if st.Username != "" {
			name = "@" + st.Username
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %d ball\n", medal, name, st.TotalScore))
	}
	if userRank > 0 {
		sb.WriteString(fmt.Sprintf("\n👤 Sizning o'rningiz: <b>#%d</b>", userRank))
	}
	if len(all) == 0 {
		sb.WriteString("<i>Hali hech kim yo'q.</i>")
	}
	b.EditMessage(sess.ChatID, msgID, sb.String(), BackKb())
}

// ============================================================
//  FEEDBACK
// ============================================================

func (b *Bot) startFeedback(sess *UserSession, msgID int) {
	sess.State = StateWaitingFeedback
	b.storage.SetSession(sess)
	b.EditMessage(sess.ChatID, msgID, "💬 <b>Fikr bildirish</b>\n\nBot haqida fikringizni yozing:\n\n<i>(/cancel bekor qilish)</i>", nil)
}

func (b *Bot) feedbackInput(sess *UserSession, text string, msg *Message) {
	if len([]rune(strings.TrimSpace(text))) < 5 {
		b.SendMessage(sess.ChatID, "❌ Juda qisqa! Kamida 5 belgi.", nil)
		return
	}
	phone := sess.Phone
	if phone == "" {
		phone = "—"
	}
	fn := sess.FirstName
	ln := sess.LastName
	un := sess.Username

	if err := b.db.SaveFeedback(sess.UserID, fn, ln, un, phone, text); err != nil {
		log.Printf("Feedback saqlash xato: %v", err)
	}

	sess.State = StateNone
	b.storage.SetSession(sess)

	// Adminlarga xabardorlik
	adminMsg := fmt.Sprintf("📨 <b>Yangi fikr!</b>\n\n👤 %s @%s\n📱 <code>%s</code>\n🆔 <code>%d</code>\n\n💬 %s",
		fullName(fn, ln), un, phone, sess.UserID, text)
	for _, adminID := range AdminIDs {
		if adminID != sess.UserID {
			b.SendMessage(adminID, adminMsg, nil)
		}
	}

	b.SendMessage(sess.ChatID, "✅ <b>Fikringiz qabul qilindi!</b>\n\nRahmat! 🙏", MainMenuKb(sess.UserID))
}

// ============================================================
//  YORDAM
// ============================================================

func buildHelp() string {
	return fmt.Sprintf(`❓ <b>Yordam — %s v%s</b>

📝 <b>Eslatmalar</b> — vazifalar ro'yxati
📖 <b>Lug'at</b> — so'zlarni yodlash
🧠 <b>Quiz</b> — bilim testi (5 kategoriya)
⏱ <b>Pomodoro</b> — 25/50 daqiqa sessiya
🎯 <b>Maqsad</b> — kunlik rejalar
📊 <b>Statistika</b> — barcha natijalar
🏆 <b>Reyting</b> — musobaqa

<b>Komandalar:</b>
/start — Ishga tushirish
/menu — Bosh menyu
/stats — Statistika
/cancel — Bekor qilish
/help — Yordam
/admin — Admin panel

<b>Muammo?</b> 💬 Fikr bildiring!`, AppName, AppVersion)
}

// ============================================================
//  ADMIN PANEL
// ============================================================

func (b *Bot) adminCallback(sess *UserSession, data string, msgID int) {
	if !isAdmin(sess.UserID) {
		return
	}
	switch data {
	case "adm_users":
		b.adminUsers(sess, msgID)
	case "adm_feedbacks":
		b.adminFeedbacks(sess, msgID)
	case "adm_stats":
		b.adminStats(sess, msgID)
	case "adm_broadcast":
		b.adminStartBroadcast(sess, msgID)
	case "adm_recent_fb":
		b.adminRecentFeedbacks(sess, msgID)
	case "adm_back":
		b.adminBack(sess, msgID)
	}
}

func (b *Bot) adminBack(sess *UserSession, msgID int) {
	total := b.db.CountUsers()
	ver := b.db.CountVerified()
	fb := b.db.CountFeedbacks()
	text := fmt.Sprintf("👨‍💼 <b>Admin Panel</b>\n\n👥 Foydalanuvchilar: <b>%d</b>\n✅ Tasdiqlangan: <b>%d</b>\n💬 Fikrlar: <b>%d</b>\n\n🕐 %s",
		total, ver, fb, time.Now().Format("02.01.2006 15:04"))
	b.EditMessage(sess.ChatID, msgID, text, AdminKb())
}

func (b *Bot) adminUsers(sess *UserSession, msgID int) {
	users, err := b.db.GetAllUsers()
	if err != nil {
		b.EditMessage(sess.ChatID, msgID, "❌ Xato yuz berdi.", AdminKb())
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 <b>Foydalanuvchilar (%d ta):</b>\n\n", len(users)))
	for i, u := range users {
		if i >= 20 {
			sb.WriteString(fmt.Sprintf("... va yana %d ta", len(users)-20))
			break
		}
		v := "❌"
		if u["verified"] == "1" {
			v = "✅"
		}
		name := fullName(u["first_name"], u["last_name"])
		un := u["username"]
		if un != "" {
			un = " @" + un
		}
		ph := u["phone"]
		if ph == "" {
			ph = "—"
		}
		t := u["joined_at"]
		if len(t) > 10 {
			t = t[:10]
		}
		sb.WriteString(fmt.Sprintf("%s <b>%d.</b> %s%s\n   📱 <code>%s</code> | 📅 %s\n\n", v, i+1, name, un, ph, t))
	}
	kb := kbInline([]InlineKeyboardButton{btn("🔙 Admin panel", "adm_back")})
	b.EditMessage(sess.ChatID, msgID, sb.String(), kb)
}

func (b *Bot) adminFeedbacks(sess *UserSession, msgID int) {
	fbs, err := b.db.GetAllFeedbacks()
	if err != nil {
		b.EditMessage(sess.ChatID, msgID, "❌ Xato yuz berdi.", AdminKb())
		return
	}
	if len(fbs) == 0 {
		b.EditMessage(sess.ChatID, msgID, "💬 <b>Fikrlar yo'q hali.</b>", AdminKb())
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("💬 <b>Barcha fikrlar (%d ta):</b>\n\n", len(fbs)))
	for i, fb := range fbs {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf("... va yana %d ta", len(fbs)-10))
			break
		}
		name := fullName(fb["first_name"], fb["last_name"])
		un := fb["username"]
		if un != "" {
			un = " @" + un
		}
		ph := fb["phone"]
		if ph == "" {
			ph = "—"
		}
		ca := fb["created_at"]
		if len(ca) > 16 {
			ca = ca[:16]
		}
		sb.WriteString(fmt.Sprintf("📨 <b>%d.</b> %s%s\n📱 <code>%s</code>\n💬 %s\n🕐 %s\n\n", i+1, name, un, ph, trunc(fb["text"], 100), ca))
	}
	kb := kbInline(
		[]InlineKeyboardButton{btn("🔄 Yangilash", "adm_feedbacks"), btn("🔙 Panel", "adm_back")},
	)
	b.EditMessage(sess.ChatID, msgID, sb.String(), kb)
}

func (b *Bot) adminRecentFeedbacks(sess *UserSession, msgID int) {
	fbs, err := b.db.GetAllFeedbacks()
	if err != nil || len(fbs) == 0 {
		b.EditMessage(sess.ChatID, msgID, "💬 <b>So'nggi fikrlar yo'q.</b>", AdminKb())
		return
	}
	var sb strings.Builder
	sb.WriteString("🆕 <b>So'nggi 5 ta fikr:</b>\n\n")
	lim := 5
	if len(fbs) < lim {
		lim = len(fbs)
	}
	for i := 0; i < lim; i++ {
		fb := fbs[i]
		name := fullName(fb["first_name"], fb["last_name"])
		un := fb["username"]
		if un != "" {
			un = " @" + un
		}
		ph := fb["phone"]
		if ph == "" {
			ph = "—"
		}
		ca := fb["created_at"]
		if len(ca) > 16 {
			ca = ca[:16]
		}
		sb.WriteString(fmt.Sprintf("📨 %s%s\n📱 <code>%s</code>\n💬 %s\n🕐 %s\n\n", name, un, ph, fb["text"], ca))
	}
	kb := kbInline(
		[]InlineKeyboardButton{btn("📋 Hammasi", "adm_feedbacks"), btn("🔙 Panel", "adm_back")},
	)
	b.EditMessage(sess.ChatID, msgID, sb.String(), kb)
}

func (b *Bot) adminStats(sess *UserSession, msgID int) {
	total := b.db.CountUsers()
	ver := b.db.CountVerified()
	fb := b.db.CountFeedbacks()
	all, _ := b.db.GetLeaderboard()
	tq, ts := 0, 0
	for _, s := range all {
		tq += s.TotalQuiz
		ts += s.TotalScore
	}
	text := fmt.Sprintf(`📊 <b>Bot umumiy statistikasi</b>

👥 <b>Foydalanuvchilar:</b>
  Jami: <b>%d</b>
  Tasdiqlangan: <b>%d</b>
  Tasdiqlanmagan: <b>%d</b>

💬 Jami fikrlar: <b>%d</b>

🧠 <b>Quiz:</b>
  Jami o'yinlar: <b>%d</b>
  Jami ball: <b>%d</b>

🕐 %s`, total, ver, total-ver, fb, tq, ts, time.Now().Format("02.01.2006 15:04"))
	kb := kbInline([]InlineKeyboardButton{btn("🔙 Admin panel", "adm_back")})
	b.EditMessage(sess.ChatID, msgID, text, kb)
}

func (b *Bot) adminStartBroadcast(sess *UserSession, msgID int) {
	sess.State = StateAdminBroadcast
	b.storage.mu.Lock()
	b.storage.setRaw(sess)
	b.storage.mu.Unlock()
	b.EditMessage(sess.ChatID, msgID, "📢 <b>Xabar yuborish</b>\n\nBarcha foydalanuvchilarga yuboriladigan xabarni kiriting:\n\n<i>(/cancel bekor qilish)</i>", nil)
}

func (b *Bot) adminBroadcastInput(sess *UserSession, text string) {
	if !isAdmin(sess.UserID) {
		sess.State = StateNone
		return
	}
	users, err := b.db.GetAllUsers()
	if err != nil {
		b.SendMessage(sess.ChatID, "❌ Foydalanuvchilar ro'yxati olinmadi.", nil)
		sess.State = StateNone
		b.storage.SetSession(sess)
		return
	}
	sess.State = StateNone
	b.storage.SetSession(sess)

	msg := "📢 <b>Admin xabari:</b>\n\n" + text
	sent, failed := 0, 0
	for _, u := range users {
		cid, _ := strconv.ParseInt(u["chat_id"], 10, 64)
		uid, _ := strconv.ParseInt(u["user_id"], 10, 64)
		if uid == sess.UserID || cid == 0 {
			continue
		}
		if _, e := b.SendMessage(cid, msg, nil); e != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(50 * time.Millisecond)
	}
	b.SendMessage(sess.ChatID, fmt.Sprintf("✅ <b>Xabar yuborildi!</b>\n\n✅ Muvaffaqiyatli: <b>%d</b>\n❌ Yuborilmadi: <b>%d</b>", sent, failed), AdminKb())
}

// ============================================================
//  BOT MAIN LOOP
// ============================================================

func (b *Bot) Run() {
	log.Printf("🤖 %s v%s ishga tushdi!", AppName, AppVersion)
	log.Printf("🗄️  Ma'lumotlar bazasi: %s", DBFile)
	log.Printf("👨‍💼 Adminlar: %v", AdminIDs)
	log.Printf("📡 Telegram API tinglanyapti...")

	offset := 0
	for {
		updates, err := b.GetUpdates(offset)
		if err != nil {
			log.Printf("❌ GetUpdates xato: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, upd := range updates {
			offset = upd.UpdateID + 1
			go b.HandleUpdate(upd)
		}
		if len(updates) == 0 {
			time.Sleep(PollInterval)
		}
	}
}

// ============================================================
//  BACKGROUND GOROUTINES
// ============================================================

func (b *Bot) runPomodoroNotifier() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sessions := b.storage.AllSessions()
		for _, sess := range sessions {
			if sess.Pomodoro == nil || !sess.Pomodoro.IsRunning {
				continue
			}
			elapsed := time.Since(sess.Pomodoro.StartTime)
			total := time.Duration(sess.Pomodoro.Duration) * time.Minute
			rem := total - elapsed
			if rem > 0 && rem <= time.Minute && rem > 30*time.Second {
				b.SendMessage(sess.ChatID, "⏰ <b>1 daqiqa qoldi!</b>\n\nYakunlashga tayyor bo'ling! 💪", nil)
				b.storage.mu.Lock()
				sess.Pomodoro.IsRunning = false
				b.storage.setRaw(sess)
				b.storage.mu.Unlock()
			} else if rem <= 0 && sess.Pomodoro.IsRunning {
				dur := sess.Pomodoro.Duration
				b.storage.mu.Lock()
				sess.Pomodoro.IsRunning = false
				sess.Stats.TotalPomodoro++
				b.storage.setRaw(sess)
				b.storage.mu.Unlock()
				go b.db.LogPomodoro(sess.UserID, dur, true)
				go b.db.SaveUserData(sess)
				b.SendMessage(sess.ChatID, fmt.Sprintf("🎉 <b>Pomodoro tugadi!</b>\n\n✅ <b>%d daqiqa</b> yakunlandi!\n☕ Dam oling!", dur), PomodoroKb())
			}
		}
	}
}

func (b *Bot) runDailyReminder() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(next.Sub(now))
		sessions := b.storage.AllSessions()
		today := time.Now().Format("2006-01-02")
		for _, sess := range sessions {
			if !sess.Verified {
				continue
			}
			pending := 0
			for _, g := range sess.Goals {
				if g.Date == today && !g.Done {
					pending++
				}
			}
			msg := fmt.Sprintf("🌅 <b>Xayrli tong, %s!</b>\n\n🔥 Streak: <b>%d kun</b>\n\n", sess.FirstName, sess.Stats.Streak)
			if pending > 0 {
				msg += fmt.Sprintf("🎯 <b>%d ta maqsad</b> kutmoqda!", pending)
			} else {
				msg += "📋 Bugun maqsad qo'shing!"
			}
			msg += "\n\n💪 Muvaffaqiyatli kun!"
			b.SendMessage(sess.ChatID, msg, MainMenuKb(sess.UserID))
		}
	}
}

func (b *Bot) runAutoSave() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sessions := b.storage.AllSessions()
		saved := 0
		for _, sess := range sessions {
			if err := b.db.SaveUserData(sess); err != nil {
				log.Printf("AutoSave xato user=%d: %v", sess.UserID, err)
			} else {
				saved++
			}
		}
		if saved > 0 {
			log.Printf("💾 AutoSave: %d ta sessiya saqlandi", saved)
		}
	}
}

// ============================================================
//  INIT & MAIN
// ============================================================

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	fmt.Printf(`
╔═══════════════════════════════════╗
║  🎓 %s v%s             ║
║  O'zbekiston o'quvchilari uchun  ║
║  SQLite DB + Verification        ║
╚═══════════════════════════════════╝
`, AppName, AppVersion)

	token := getBotToken()
	if token == "" || token == "YOUR_BOT_TOKEN_HERE" {
		log.Fatal(`❌ Bot token topilmadi!

1. Muhit o'zgaruvchisi:
   export TELEGRAM_BOT_TOKEN="sizning_tokeningiz"

2. Yoki BotToken konstantasini o'zgartiring

Token: @BotFather → /newbot`)
	}

	db, err := NewDatabase(DBFile)
	if err != nil {
		log.Fatalf(`❌ Ma'lumotlar bazasini ochib bo'lmadi: %v

Eslatma: go-sqlite3 uchun CGO kerak:
  CGO_ENABLED=1 go run main.go

O'rnatish:
  go get github.com/mattn/go-sqlite3`, err)
	}
	log.Printf("✅ Ma'lumotlar bazasi tayyor: %s", DBFile)

	bot := NewBot(token, db)

	go bot.runPomodoroNotifier()
	go bot.runDailyReminder()
	go bot.runAutoSave()

	bot.Run()
}