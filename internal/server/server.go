package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Vill785/go_final_project/internal/utils"
)

// Описывает задачу из таблицы scheduler
type Task struct {
	ID      int    `json:"id,string"` // возвращаем тег ,string, чтобы id корректно декодировался из JSON
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

// Объединяет данные для работы сервера и БД
type Server struct {
	db     *sql.DB
	addr   string
	webDir string
}

func NewServer(db *sql.DB, addr, webDir string) *Server {
	return &Server{
		db:     db,
		addr:   addr,
		webDir: webDir,
	}
}

// Главный роутер для /api/task – различает методы GET, POST, PUT
func (s *Server) handleTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getTaskByIDHandler(w, r)
	case http.MethodPost:
		s.createTaskHandler(w, r)
	case http.MethodPut:
		s.updateTaskHandler(w, r)
	case http.MethodDelete:
		// Реализация удаления задачи по id
		idParam := r.URL.Query().Get("id")
		if idParam == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
			return
		}
		id, err := strconv.Atoi(idParam)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Неверный формат идентификатора"})
			return
		}
		res, err := s.db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request method"})
	}
}

// GET /api/task?id=<идентификатор>
// Возвращает JSON-объект со всеми полями задачи или ошибку, если задача не найдена
func (s *Server) getTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(map[string]string{"error": "Неверный формат идентификатора"})
		return
	}

	row := s.db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`, id)
	var task Task
	err = row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		return
	}

	response := map[string]string{
		"id":      strconv.Itoa(task.ID),
		"date":    task.Date,
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(response)
}

// POST /api/task – создание задачи (оставляем без изменений)
func (s *Server) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "ошибка десериализации JSON"})
		return
	}

	if strings.TrimSpace(task.Title) == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан заголовок задачи"})
		return
	}

	now := time.Now().In(time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStr := today.Format("20060102")

	if strings.TrimSpace(task.Repeat) != "" {
		if task.Repeat != "y" && !strings.HasPrefix(task.Repeat, "d ") {
			json.NewEncoder(w).Encode(map[string]string{"error": "неподдерживаемый формат repeat"})
			return
		}
		if strings.HasPrefix(task.Repeat, "d ") {
			parts := strings.SplitN(task.Repeat, " ", 2)
			if len(parts) != 2 {
				json.NewEncoder(w).Encode(map[string]string{"error": "некорректный формат: не указан интервал дней"})
				return
			}
			interval, err := strconv.Atoi(parts[1])
			if err != nil || interval <= 0 || interval > 400 {
				json.NewEncoder(w).Encode(map[string]string{"error": "некорректный интервал дней"})
				return
			}
		}
	}

	if strings.TrimSpace(task.Date) == "" {
		task.Date = todayStr
	} else {
		parsedDate, err := time.Parse("20060102", task.Date)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Дата представлена в неверном формате"})
			return
		}
		if parsedDate.Before(today) {
			if strings.TrimSpace(task.Repeat) != "" {
				nextDate, err := utils.NextDate(today, task.Date, task.Repeat)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}
				task.Date = nextDate
			} else {
				task.Date = todayStr
			}
		}
	}

	res, err := s.db.Exec(
		`INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`,
		task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"id": strconv.FormatInt(id, 10)})
}

// PUT /api/task?id=<идентификатор> – редактирование задачи
// Если в теле JSON не указан id, берется из query-параметра.
// При успешном обновлении возвращается пустой JSON {}.
func (s *Server) updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "ошибка десериализации JSON"})
		return
	}

	// Если в теле не пришёл id, пробуем взять его из query-параметра
	if task.ID == 0 {
		idParam := r.URL.Query().Get("id")
		if idParam != "" {
			if id, err := strconv.Atoi(idParam); err == nil {
				task.ID = id
			}
		}
	}
	if task.ID == 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
		return
	}

	if strings.TrimSpace(task.Title) == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан заголовок задачи"})
		return
	}

	now := time.Now().In(time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStr := today.Format("20060102")

	if strings.TrimSpace(task.Repeat) != "" {
		if task.Repeat != "y" && !strings.HasPrefix(task.Repeat, "d ") {
			json.NewEncoder(w).Encode(map[string]string{"error": "неподдерживаемый формат repeat"})
			return
		}
		if strings.HasPrefix(task.Repeat, "d ") {
			parts := strings.SplitN(task.Repeat, " ", 2)
			if len(parts) != 2 {
				json.NewEncoder(w).Encode(map[string]string{"error": "некорректный формат: не указан интервал дней"})
				return
			}
			interval, err := strconv.Atoi(parts[1])
			if err != nil || interval <= 0 || interval > 400 {
				json.NewEncoder(w).Encode(map[string]string{"error": "некорректный интервал дней"})
				return
			}
		}
	}

	if strings.TrimSpace(task.Date) == "" {
		task.Date = todayStr
	} else {
		parsedDate, err := time.Parse("20060102", task.Date)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Дата представлена в неверном формате"})
			return
		}
		if parsedDate.Before(today) {
			if strings.TrimSpace(task.Repeat) != "" {
				nextDate, err := utils.NextDate(today, task.Date, task.Repeat)
				if err != nil {
					json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
					return
				}
				task.Date = nextDate
			} else {
				task.Date = todayStr
			}
		}
	}

	res, err := s.db.Exec(
		`UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`,
		task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if rowsAffected == 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{})
}

// GET /api/tasks – возвращает список задач
func (s *Server) getTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.db.Query(`SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date`)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	tasks := make([]map[string]string, 0)
	for rows.Next() {
		var id int
		var date, title, comment, repeat string
		if err := rows.Scan(&id, &date, &title, &comment, &repeat); err != nil {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		task := map[string]string{
			"id":      strconv.Itoa(id),
			"date":    date,
			"title":   title,
			"comment": comment,
			"repeat":  repeat,
		}
		tasks = append(tasks, task)
	}

	response := map[string]interface{}{
		"tasks": tasks,
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"error": "ошибка кодирования JSON"}`, http.StatusInternalServerError)
	}
}

func (s *Server) handleTaskDone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Не указан идентификатор"})
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Неверный формат идентификатора"})
		return
	}

	switch r.Method {
	case http.MethodPost:
		var task Task
		err := s.db.QueryRow(`SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`, id).
			Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
			return
		}

		if strings.TrimSpace(task.Repeat) == "" {
			// Одноразовая задача – удаляем запись
			res, err := s.db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil || rowsAffected == 0 {
				json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
				return
			}
			// Возвращаем пустой JSON {}
			json.NewEncoder(w).Encode(map[string]interface{}{})
		} else {
			// Периодическая задача – рассчитываем следующую дату
			now := time.Now().In(time.UTC)
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			nextDate, err := utils.NextDate(today, task.Date, task.Repeat)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			res, err := s.db.Exec(`UPDATE scheduler SET date = ? WHERE id = ?`, nextDate, id)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil || rowsAffected == 0 {
				json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
				return
			}
			// Возвращаем пустой JSON {}
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	case http.MethodDelete:
		// DELETE-запрос – удаляем задачу
		res, err := s.db.Exec(`DELETE FROM scheduler WHERE id = ?`, id)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			json.NewEncoder(w).Encode(map[string]string{"error": "Задача не найдена"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request method"})
	}
}

// Регистрируем маршруты и запускаем сервер
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API-обработчики
	mux.HandleFunc("/api/task", s.handleTask)
	mux.HandleFunc("/api/tasks", s.getTasksHandler)
	mux.HandleFunc("/api/tasks/", s.getTasksHandler)
	mux.HandleFunc("/api/nextdate", s.handleNextDate)
	mux.HandleFunc("/api/task/done", s.handleTaskDone)
	mux.HandleFunc("/api/task/done/", s.handleTaskDone)

	// Файловый сервер для остальных маршрутов
	fs := http.FileServer(http.Dir(s.webDir))
	mux.Handle("/", fs)

	log.Printf("Сервер запущен на http://localhost%s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// GET /api/nextdate – оставляем без изменений
func (s *Server) handleNextDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	nowStr := query.Get("now")
	dateStr := query.Get("date")
	repeat := query.Get("repeat")
	if nowStr == "" || dateStr == "" || repeat == "" {
		return
	}
	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "Неверный формат now", http.StatusBadRequest)
		return
	}
	nextDate, err := utils.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}
