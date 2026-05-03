# BilimQuiz — CLAUDE.md

Интерактивдүү билим берүү платформасы (Kahoot аналогу). Кыргыз, Орус, Англис тилдеринде.

## Технологиялык стек

- **Backend**: Go 1.26, `chi` router, `pgx/v5` (PostgreSQL), `gorilla/websocket`, `gorilla/sessions`
- **Frontend**: HTML Templates + Tailwind CSS CDN + Alpine.js + HTMX
- **Database**: PostgreSQL 16
- **Auth**: Google OAuth2 (`golang.org/x/oauth2`)
- **AI**: Gemini API (HTTP direct, `gemini-2.0-flash`)
- **QR**: `skip2/go-qrcode`
- **DevOps**: Docker Compose

## Негизги командалар

```bash
go run ./cmd/server          # сервер иштетүү (порт 8080)
go build ./...               # компиляция текшерүү
GOFLAGS=-mod=mod go build ./... # go.sum жетишсиз болсо
docker compose up -d         # PostgreSQL баштоо
docker compose up -d --build # баарын баштоо
```

## Папка структурасы

```
cmd/server/main.go           # Entrypoint — роутер, middleware, бардыгын байлоо
internal/
  auth/
    session.go               # Cookie session: SetTeacherID, GetTeacherID,
                             #   SetRedirectAfterLogin, GetRedirectAfterLogin, ClearRedirectAfterLogin
    oauth.go                 # Google OAuth2: GetAuthURL, ExchangeCode → GoogleUser
  db/
    db.go                    # pgxpool.New() + Migrate() runner (migrations/*.sql)
  repository/
    models.go                # Бардык struct'тар: Teacher, Game (ShareToken кирет), Question, Session...
    teacher.go               # Upsert (Google login), GetByID, GetStats, UpdateGeminiKey
    game.go                  # CRUD + QuestionCount JOIN + GetByShareToken
    question.go              # CRUD + ReplaceAnswers (delete-insert)
    session.go               # Create, GetByPin, Snapshot, AddPlayer, SavePlayerAnswer, Leaderboard
  game/
    hub.go                   # Global Hub — rooms map[pin]*Room, GeneratePin()
    room.go                  # Room: register/unregister channels, broadcastAll, kickPlayer
    client.go                # WebSocket Client: ReadPump / WritePump goroutines
    engine.go                # startGame → runAsync → handleAnswer → упай эсептөө
    types.go                 # GameState, Message, SnapshotQuestion, AnswerMsg
  handlers/
    render.go                # LoadTemplates + Render(w,r,name,data) + funcMap (t, appName, inc, fmtTime, ms2s, jsonQuestions, jsonPlayers)
    auth.go                  # GoogleLogin (?next= колдоосу), GoogleCallback (redirect_after_login), Logout
    teacher.go               # Dashboard, SaveGeminiKey
    game.go                  # NewGame, CreateGame, EditGame, UpdateGame, DeleteGame, AddQuestion, UpdateQuestion, DeleteQuestion
    student.go               # JoinPage, CheckPin, LobbyPage, JoinLobby, PlayPage, ResultPage
    play.go                  # StartSession (PIN генерация + Snapshot), LobbyPage, LobbyPlayers, MonitorPage, PodiumPage
    shared.go                # SharedHandler: GamePage (GET /shared/{token}), StartSession (POST /shared/{token}/start)
    ws.go                    # TeacherLobbyWS, PlayerWS (WebSocket upgrade)
    history.go               # List, SessionDetail, PlayerDetail
    ai.go                    # Generate (Gemini API → DB сактоо)
    upload.go                # UploadPlayerImage (Base64 data URI → player_images/)
  middleware/
    auth.go                  # RequireAuth — авторизациясыз болсо URL session'го сактап /auth/google'га redirect
    lang.go                  # Lang — query→cookie→Accept-Language→"ky" + cookie сет
    logger.go                # HTTP request logger
  i18n/
    i18n.go                  # Load(dir), T(lang,key), WithLang(ctx), FromContext(ctx), DetectLang
  qr/
    qr.go                    # GET /qr/{pin} → PNG image
  ai/
    gemini.go                # GenerateQuestions(ctx, apiKey, topic, count) → []GeneratedQuestion
migrations/
  001_init.sql               # teachers, games, questions, answers + update_updated_at trigger
  002_sessions.sql           # game_sessions, session_questions_snapshot, session_answers_snapshot, session_players, player_answers
  003_share_token.sql        # games.share_token UUID NOT NULL DEFAULT gen_random_uuid()
locales/
  ky.json / ru.json / en.json  # 64 UI текст ачкычы (share_game, share_link, share_copy, share_copied, shared_start_session, shared_logged_in кирет)
templates/
  landing.html               # Башкы бет: PIN форма + Google login + lang modal
  dashboard.html             # Мугалим кабинети: stats + games table + Gemini key + 🔗 Share dropdown
  game_builder.html          # Оюн редактору: сол панел (суроолор) + форма (Alpine.js)
  join.html                  # Окуучу: PIN киргизүү
  lobby_student.html         # Окуучу: ат + аватар тандоо → күтүү залы
  play_student.html          # Окуучу: суроо + жооп баскычтары + таймер (WebSocket)
  result_student.html        # Окуучу: жыйынтык + breakdown
  lobby_teacher.html         # Мугалим: PIN/QR + оюнчулар тизмеси + баштоо
  monitor_teacher.html       # Мугалим: "тепкич" анимациясы (WebSocket live)
  podium.html                # Мугалим: 1-2-3 пьедестал + калгандар
  history_list.html          # Тарых: сессиялар тизмеси
  history_session.html       # Тарых: сессиянын оюнчулар рейтинги
  history_player.html        # Тарых: жеке окуучунун жооп аналитикасы
  shared_game.html           # Бөлүшүлгөн оюн: аталышы, автору, суроолор саны + Сессия баштоо
static/
  css/app.css                # Tailwind utility класстары (btn-primary, card, input, label...)
  js/app.js                  # compressAndUpload (Canvas API), createWS (WebSocket helper)
  audio/lobby.mp3            # BGM — өзүңүз кошуңуз (жок болсо тынч иштейт)
player_images/               # Окуучулардын аватарлары (Base64→JPEG)
```

## URL маршруттары

### Public (авторизациясыз)
| URL | Метод | Аракет |
|-----|-------|--------|
| `/` | GET | Landing page |
| `/auth/google` | GET | Google OAuth redirect (`?next=` param колдоосу бар) |
| `/auth/google/callback` | GET | OAuth callback → session'дон redirect URL окуп redirect |
| `/logout` | GET | Session тазалоо |
| `/lang/{ky\|ru\|en}` | GET | Тил өзгөртүү cookie |
| `/join` | GET | PIN форма |
| `/join/check` | POST | PIN валидация |
| `/lobby/{pin}` | GET | Окуучу lobby |
| `/lobby/{pin}/join` | POST | Оюнга кошулуу → JSON `{player_id}` |
| `/play/{pin}/{player_id}` | GET | Окуучу оюн экраны |
| `/result/{player_id}` | GET | Окуучу жыйынтыгы |
| `/ws/player/{pin}/{player_id}` | WS | Окуучу WebSocket |
| `/upload/avatar` | POST | Base64 сүрөт жүктөө |
| `/qr/{pin}` | GET | PNG QR код |

### Protected (мугалим, session cookie керек)
| URL | Метод | Аракет |
|-----|-------|--------|
| `/dashboard` | GET | Статистика + оюндар |
| `/games/new` | GET/POST | Жаңы оюн |
| `/games/{id}/edit` | GET/POST | Оюн редактору |
| `/games/{id}/delete` | POST | Оюн өчүрүү |
| `/games/{id}/questions` | POST | Суроо кошуу |
| `/questions/{qid}/update` | POST | Суроо жаңыртуу |
| `/questions/{qid}/delete` | POST | Суроо өчүрүү |
| `/play/{id}` | GET | Сессия түзүү + PIN генерация |
| `/teacher/lobby/{session_id}` | GET | Мугалим lobby |
| `/teacher/lobby/{session_id}/players` | GET | JSON оюнчулар |
| `/teacher/monitor/{session_id}` | GET | Live monitor |
| `/teacher/podium/{session_id}` | GET | Жыйынтык |
| `/ws/teacher/{session_id}` | WS | Мугалим WebSocket |
| `/history` | GET | Сессиялар тизмеси |
| `/history/{id}` | GET | Сессия деталы |
| `/history/{id}/player/{player_id}` | GET | Окуучу аналитикасы |
| `/api/ai/generate` | POST | Gemini AI суроо генерациясы |
| `/shared/{token}` | GET | Бөлүшүлгөн оюн барагы |
| `/shared/{token}/start` | POST | Кирген мугалим сессия баштайт |

## WebSocket протоколу

### Мугалим ← → Сервер
```json
// Мугалим → сервер
{"type": "start_game"}
{"type": "kick", "player_id": 42}

// Сервер → мугалим
{"type": "lobby_update", "player_count": 5, "players": [...]}
{"type": "player_progress", "payload": {"player_id": 1, "questions_answered": 3, "total_score": 2400}}
{"type": "game_finished", "payload": {"total_players": 12}}
```

### Окуучу ← → Сервер
```json
// Сервер → окуучу
{"type": "game_start"}
{"type": "question", "index": 0, "total": 10, "question": {...}}
{"type": "answer_result", "is_correct": true, "earned_points": 850, "total_score": 850, "answers": [...]}
{"type": "game_over", "final_score": 3200}
{"type": "kicked"}

// Окуучу → сервер
{"type": "answer", "answer": "Париж", "time_taken_ms": 4200, "question_id": 17}
```

## Маалымат базасы

### Негизги таблицалар (өзгөртүлөт)
- `teachers` — google_id, email, name, avatar_url, language, gemini_key
- `games` — teacher_id, title, description, **share_token** (UUID, auto-generated, unique)
- `questions` — game_id, position, content, image_url, youtube_url, youtube_start/end, time_limit, score_type (`dynamic`\|`static`), static_score
- `answers` — question_id, text, is_correct

### Тарых таблицалары (immutable snapshot)
- `game_sessions` — pin_code, status (`waiting`\|`active`\|`finished`), total_players
- `session_questions_snapshot` — оюн башталганда суроолордун көчүрмөсү
- `session_answers_snapshot` — жооп вариянттарынын көчүрмөсү
- `session_players` — nickname, avatar, final_score
- `player_answers` — selected_answer_text, is_correct, earned_points, time_taken_ms

### Миграция
`db.Migrate()` сервер старттанганда `schema_migrations` таблица аркылуу автоматтык.

## Упай эсептөө логикасы (`engine.go`)

```
dynamic: earned = round(1000 × max(0.1, 1 - timeSec/timeLimit))
static:  earned = question.static_score (эгер туура болсо)
```

## Template функциялары (`render.go`)

| Функция | Колдонуу |
|---------|----------|
| `t .Lang "key"` | Котормо: `{{t .Lang "login"}}` |
| `appName` | `.env APP_NAME` же `"BilimQuiz"` |
| `inc $i` | `$i + 1` (таблица номерлоо) |
| `fmtTime $t` | `02.01.2006 15:04` форматы |
| `ms2s $ms` | Миллисекунддан секундга: `"4.2"` |
| `jsonQuestions .Questions` | `[]Question → template.JS` (game_builder) |
| `jsonPlayers .Players .Progress .Scores` | `[]SessionPlayer + maps → template.JS` (monitor) |

## i18n

- Тил аныктоо: `?lang=` query → `lang` cookie → `Accept-Language` header → `"ky"` default
- Өзгөртүү: `GET /lang/{ky|ru|en}` → cookie сет → referer'га redirect
- Котормо файлдары: `locales/ky.json`, `ru.json`, `en.json`
- Мугалим тили маалымат базасында да сакталат (`teachers.language`)

## .env өзгөрмөлөрү

```env
APP_NAME=BilimQuiz          # Сайт аталышы (logo жана title)
APP_PORT=8080
DB_URL=postgres://...       # pgx connection string
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
SESSION_SECRET=...          # Кем дегенде 32 символ
GEMINI_API_KEY=...          # Опционалдуу (мугалим өзү да киргизе алат)
PLAYER_IMAGES_DIR=./player_images
```

## Маанилүү архитектуралык чечимдер

1. **Snapshot pattern** — оюн башталганда `session_questions_snapshot` + `session_answers_snapshot` таблицаларына маалымат жазылат. Тарыхтагы баллдар суроо өзгөртүлгөндөн кийин да бузулбайт.

2. **Асинхрондук gameplay** — Ар бир окуучу өзүнүн ылдамдыгы менен жооп берет. Мугалим башкаларды күтпөстөн монитордон ар кимдин прогрессин көрөт.

3. **WebSocket Hub** — `game.Global` — жалгыз hub, `map[pin]*Room`. Мугалим `/ws/teacher/{session_id}` аркылуу room түзөт; окуучулар `/ws/player/{pin}/{player_id}` аркылуу кошулат.

4. **Canvas компрессия** — Окуучунун аватар сүрөтү JavaScript Canvas API аркылуу client-side'да 120×120px'ге кысылат, андан кийин Base64 POST жасалат. Сервер тарапта `/upload/avatar` эндпоинти JPEG файлды `player_images/` папкасына сактайт.

5. **Тил middleware** — Ар бир request'те `middleware.Lang` тилди аныктап, `context.WithValue` аркылуу тил кодун өткөрөт. Бардык шаблондор `{{t .Lang "key"}}` аркылуу которулат.

6. **Post-login redirect** — `RequireAuth` middleware авторизациясыз кирүүдө `r.RequestURI` session'го (`redirect_after_login` ачкычы) сактайт да `/auth/google`'га redirect кылат. `GoogleCallback` ийгиликтүү логин кийин ошол URL'га кайтат, болбосо `/dashboard`'га.

7. **Game share link** — Ар бир оюндун `share_token` UUID'си бар (auto-generated). Dashboard'дан `🔗` баскычы менен `http://host/shared/{token}` шилтемесин clipboard'го көчүрүүгө болот. Шилтеме менен кирген адам авторизациядан өткөндөн кийин оюнду өзүнүн атынан сессия катары баштай алат.
