package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/paran0iaa/go_final/internal/db"
	"github.com/paran0iaa/go_final/internal/task"
	"github.com/paran0iaa/go_final/internal/utils"
)

const layout = "20060102"

func RegisterAPIRoutes(r *chi.Mux) {
	r.Get("/api/nextdate", HandleNextDate)
	r.Post("/api/task", HandleAddTask)
	r.Get("/api/tasks", task.Tasks)
	r.Get("/api/task", getTaskHandler)
	r.Put("/api/task", updateTaskHandler)
	r.Post("/api/task/done", handleTaskDone)
	r.Delete("/api/task", handleTaskDelete)
}

func HandleNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeatStr := r.FormValue("repeat")

	now, err := time.Parse(layout, nowStr)
	if err != nil {
		http.Error(w, "Недопустимый формат даты", http.StatusBadRequest)
		return
	}

	nextDate, err := utils.NextDate(now, dateStr, repeatStr, "nextdate")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}

func HandleAddTask(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		task.AddTask(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "отсутствует id"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "недопустимый параметр id"})
		return
	}

	foundTask, err := db.GetTaskByID(id)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "задача не найдена"})
		return
	}

	response := map[string]string{
		"id":      strconv.FormatInt(foundTask.ID, 10),
		"date":    foundTask.Date,
		"title":   foundTask.Title,
		"comment": foundTask.Comment,
		"repeat":  foundTask.Repeat,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

var req struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат JSON"})
		return
	}

	if req.ID == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Недопустимый формат идентификатора"})
		return
	}

	if req.Date == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Дата не может быть пустой"})
		return
	}

	_, err = time.Parse(layout, req.Date)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат даты"})
		return
	}

	if req.Title == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Заголовок не может быть пустым"})
		return
	}

	updatedTask := db.Task{
		ID:      id,
		Date:    req.Date,
		Title:   req.Title,
		Comment: req.Comment,
		Repeat:  req.Repeat,
	}

	taskErr := db.UpdateTask(updatedTask)
	if taskErr != nil {
		if errors.Is(taskErr, db.ErrTaskNotFound) {
			utils.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
		} else {
			utils.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка обновления задачи"})
		}
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{})
}

func handleTaskDone(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Отсутствует идентификатор задачи"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Недопустимый формат идентификатора"})
		return
	}

	foundTask, err := db.GetTaskByID(id)
	if err != nil {
		if errors.Is(err, db.ErrTaskNotFound) {
			utils.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
			return
		}
		utils.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка при получении задачи"})
		return
	}

	if foundTask.Repeat == "" {
		err = db.DeleteTask(id)
		if err != nil {
			utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Ошибка удаления задачи"})
			return
		}
	} else {
		now := time.Now()
		nextDate, err := utils.NextDate(now, foundTask.Date, foundTask.Repeat, "done")
		if err != nil {
			utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Ошибка вычисления следующей даты"})
			return
		}

		foundTask.Date = nextDate
		err = db.UpdateTask(foundTask)
		if err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка обновления задачи"})
			return
		}
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{})
}

func handleTaskDelete(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Отсутствует идентификатор задачи"})
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Недопустимый формат идентификатора"})
		return
	}

	err = db.DeleteTask(id)
	if err != nil {
		if errors.Is(err, db.ErrTaskNotFound) {
			utils.RespondWithJSON(w, http.StatusNotFound, map[string]string{"error": "Задача не найдена"})
			return
		}
		utils.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка удаления задачи"})
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{})
}
