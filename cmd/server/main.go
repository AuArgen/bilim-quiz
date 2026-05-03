package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"bilim-quiz/internal/auth"
	"bilim-quiz/internal/db"
	"bilim-quiz/internal/handlers"
	"bilim-quiz/internal/i18n"
	"bilim-quiz/internal/middleware"
	"bilim-quiz/internal/onboarding"
	"bilim-quiz/internal/qr"
	"bilim-quiz/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, using environment variables")
	}

	auth.InitStore()
	auth.InitOAuth()

	if err := i18n.Load("./locales"); err != nil {
		log.Fatalf("load i18n: %v", err)
	}

	ctx := context.Background()
	pool, err := db.New(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "./migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := handlers.LoadTemplates("./templates"); err != nil {
		log.Fatalf("load templates: %v", err)
	}

	// Repos
	teacherRepo := repository.NewTeacherRepo(pool)
	gameRepo := repository.NewGameRepo(pool)
	questionRepo := repository.NewQuestionRepo(pool)
	sessionRepo := repository.NewSessionRepo(pool)

	// Handlers
	ob := onboarding.Deps{Games: gameRepo, Questions: questionRepo}
	authH    := handlers.NewAuthHandler(teacherRepo, ob)
	teacherH := handlers.NewTeacherHandler(teacherRepo, gameRepo)
	historyH := handlers.NewHistoryHandler(sessionRepo, gameRepo, teacherRepo)
	aiH      := handlers.NewAIHandler(teacherRepo, questionRepo)
	gameH    := handlers.NewGameHandler(gameRepo, questionRepo)
	studentH := handlers.NewStudentHandler(sessionRepo, questionRepo)
	playH    := handlers.NewPlayHandler(gameRepo, sessionRepo, questionRepo)
	wsH      := handlers.NewWSHandler(sessionRepo, questionRepo)
	sharedH  := handlers.NewSharedHandler(gameRepo, sessionRepo, questionRepo, teacherRepo)

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Lang)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	r.Handle("/player_images/*", http.StripPrefix("/player_images/", http.FileServer(http.Dir("./player_images"))))

	// QR code
	r.Get("/qr/{pin}", qr.Handler)

	// Upload (public — student avatar)
	r.Post("/upload/avatar", handlers.UploadPlayerImage)

	// Student routes (public)
	r.Get("/join", studentH.JoinPage)
	r.Post("/join/check", studentH.CheckPin)
	r.Get("/lobby/{pin}", studentH.LobbyPage)
	r.Post("/lobby/{pin}/join", studentH.JoinLobby)
	r.Get("/play/{pin}/{player_id}", studentH.PlayPage)
	r.Get("/result/{player_id}", studentH.ResultPage)

	// Public routes
	r.Get("/", authH.LoginPage)
	r.Get("/auth/google", authH.GoogleLogin)
	r.Get("/auth/google/callback", authH.GoogleCallback)
	r.Get("/logout", authH.Logout)

	// Set language
	r.Get("/lang/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		http.SetCookie(w, &http.Cookie{
			Name: "lang", Value: code, Path: "/", MaxAge: 86400 * 365,
		})
		ref := r.Referer()
		if ref == "" {
			ref = "/"
		}
		http.Redirect(w, r, ref, http.StatusFound)
	})

	// Protected routes (teacher)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)

		r.Get("/dashboard", teacherH.Dashboard)
		r.Post("/dashboard/gemini-key", teacherH.SaveGeminiKey)

		r.Get("/games/new", gameH.NewGame)
		r.Post("/games/new", gameH.CreateGame)
		r.Get("/games/{id}/edit", gameH.EditGame)
		r.Post("/games/{id}/edit", gameH.UpdateGame)
		r.Post("/games/{id}/delete", gameH.DeleteGame)

		r.Post("/games/{id}/questions", gameH.AddQuestion)
		r.Post("/questions/{qid}/update", gameH.UpdateQuestion)
		r.Post("/questions/{qid}/delete", gameH.DeleteQuestion)

		// Play flow (teacher)
		r.Get("/play/{id}", playH.StartSession)
		r.Get("/teacher/lobby/{session_id}", playH.LobbyPage)
		r.Get("/teacher/lobby/{session_id}/players", playH.LobbyPlayers)
		r.Get("/teacher/monitor/{session_id}", playH.MonitorPage)
		r.Get("/teacher/podium/{session_id}", playH.PodiumPage)

		// WebSocket (teacher)
		r.Get("/ws/teacher/{session_id}", wsH.TeacherLobbyWS)

		// History
		r.Get("/history", historyH.List)
		r.Get("/history/{id}", historyH.SessionDetail)
		r.Get("/history/{id}/player/{player_id}", historyH.PlayerDetail)

		// AI
		r.Post("/api/ai/generate", aiH.Generate)
	})

	// Shared game routes (require auth)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Get("/shared/{token}", sharedH.GamePage)
		r.Post("/shared/{token}/start", sharedH.StartSession)
	})

	// WebSocket (student — public, no auth)
	r.Get("/ws/player/{pin}/{player_id}", wsH.PlayerWS)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 BilimQuiz started on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}
