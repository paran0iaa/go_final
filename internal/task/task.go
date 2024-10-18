package task

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/paran0iaa/go_final/internal/utils"

	"github.com/paran0iaa/go_final/internal/db"
)

const layout = "20060102"

type AddTaskRequest struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type AddTaskResponse struct {
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

func AddTask(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "не удалось прочитать тело запроса"})
		return
	}
	defer r.Body.Close()

	var req AddTaskRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "неверный формат JSON"})
		return
	}

	// проверка обязательного поля title
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		utils.RespondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "не указан заголовок задачи"})
		return
	}

	// обработка поля date
	var taskDate time.Time
	now := time.Now()

	if strings.TrimSpace(req.Date) == "" {
		req.Date = now.Format(layout)
	}

	taskDate, err = time.Parse(layout, req.Date)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "дата указана в неверном формате"})
		return
	}

	if taskDate.Before(now) {
		if strings.TrimSpace(req.Repeat) == "" {
			taskDate = now
		} else {
			nextDateStr, err := utils.NextDate(now, req.Date, req.Repeat, "add")
			if err != nil {
				utils.RespondWithJSON(w, http.StatusBadRequest, AddTaskResponse{Error: "неверное правило повторения"})
				return
			}
			taskDate, _ = time.Parse(layout, nextDateStr)
		}
	}

	// создание объекта задачи
	newTask := db.Task{
		Date:    taskDate.Format(layout),
		Title:   req.Title,
		Comment: req.Comment,
		Repeat:  req.Repeat,
	}

	// добавление задачи в базу данных
	id, err := db.AddTask(newTask)
	if err != nil {
		utils.RespondWithJSON(w, http.StatusInternalServerError, AddTaskResponse{Error: err.Error()})
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, AddTaskResponse{ID: fmt.Sprintf("%d", id)})
}

type TaskResponseItem struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TasksResponse struct {
	Tasks []TaskResponseItem `json:"tasks"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func Tasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := db.GetUpcomingTasks()
	if err != nil {
		log.Printf("Ошибка получения задач: %w", err)
		utils.RespondWithJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	response := TasksResponse{Tasks: []TaskResponseItem{}}

	for _, t := range tasks {
		taskItem := TaskResponseItem{
			ID:      fmt.Sprintf("%d", t.ID),
			Date:    t.Date,
			Title:   t.Title,
			Comment: t.Comment,
			Repeat:  t.Repeat,
		}
		response.Tasks = append(response.Tasks, taskItem)
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}
