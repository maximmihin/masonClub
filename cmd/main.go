package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"log"
	"masonClub/app/storage"
	"masonClub/app/use_cases"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
		log.Print(err)
	}
	port := os.Getenv("PORT")
	db := os.Getenv("DB")
	jwtSecret := os.Getenv("JWT_SECRET")

	run(port, db, jwtSecret)
}

func run(port, db, jwtSecret string) {

	store, err := storage.New(db)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/join", JoinToMasons(store, jwtSecret))
	http.HandleFunc("/masons_list", CheckJwtToken(jwtSecret, store, GetMasonsList(store)))

	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("не запустился http сервер: %s", err)
	}

}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	text := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Надежный клуб масонов</title>
</head>
<body>
    <h1>Привет! Это надежный клуб масонов</h1>
    <p>
		Ты можешь присоединиться к нам! Для этого тебе нужно отправить http запрос на /join<br>
		cо своим псевдонимом в параметре pseudonym, например:<br>
		<br>		
		localhost:8080/join?pseudonym=anonimus<br>
		<br>
		Если этот псевдоним не занят, ты получишь JWT токен. Без этого токена ты не сможешь увидеть список членов клуба.<br>
		Чтобы увидеть других участников помести этот токен в Authorization header (bearer token) и отправь запрос на /masons_list, например:<br>
		<br>		
		localhost:8080/masons_list<br>
		<br>
		После этого ты сможешь узнать, кто уже состоит в нашем клубе<br>
	</p>
</body>
</html>
`
	fmt.Fprintf(w, text)
}

// стать новым масоном (зарегистрироваться как масон)
func JoinToMasons(store *storage.Store, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hasPseudonym := r.URL.Query().Has("pseudonym")
		if !hasPseudonym {
			sendError(w, "Нужно указать параметр pseudonym", http.StatusBadRequest)
			return
		}
		pseudonym := r.URL.Query().Get("pseudonym")

		mason, err := use_cases.InitiationIntoTheMasons(store, jwtSecret, pseudonym, time.Now())
		if err != nil {
			if errors.Is(err, use_cases.ErrMasonAlreadyInitiation) {
				sendError(w, err.Error(), http.StatusBadRequest)
				return
			}
			sendError(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Token   string `json:"bearer_token"`
		}{
			Status:  fmt.Sprintf("Успех"),
			Message: fmt.Sprintf("%s, теперь вы масон!", mason.Pseudonym),
			Token:   mason.JwtToken,
		})
		return
	}
}

// todo прикрутить функционал пагинции

// получить список масонов
func GetMasonsList(store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masons, err := use_cases.GetListMasons(store)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(struct {
					Status  string `json:"status"`
					Message string `json:"message"`
				}{
					Status:  fmt.Sprintf("Успех"),
					Message: "сейчас у нас нет ни одного масона :(",
				})
			}
			sendError(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Status           string   `json:"status"`
			MasonsPseudonyms []string `json:"our_masons"`
		}{
			Status:           "Успех",
			MasonsPseudonyms: masons,
		})

	}
}

func CheckJwtToken(secretKey string, store *storage.Store, handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		authHeader := request.Header.Get("Authorization")
		if authHeader == "" {
			sendError(writer, "в хедере нет jwt токена", http.StatusUnauthorized)
			return
		}

		gotToken := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(gotToken, func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(secretKey), nil
		})
		if err != nil || !token.Valid {
			sendError(writer, "невалидный токен", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			sendError(writer, "в токене нет playground", http.StatusUnauthorized)
			return
		}

		pseudo, ok := claims["sub"]
		if !ok {
			sendError(writer, "в токене нет псевдонима масона", http.StatusUnauthorized)
			return
		}

		pseudonym, ok := pseudo.(string)
		if !ok {
			sendError(writer, "в токене неправильный pseudonym (не строка)", http.StatusUnauthorized)
			return
		}

		// todo шляпа - ? добавить слой юзкейса
		// мб сделать DI контейнер
		_, err = store.GetMasonByPseudonym(pseudonym)
		if err != nil {
			if errors.Is(err, storage.ErrMasonNotExist) {
				sendError(writer, "у нас нет такого масона! Как ты подделал токен?", http.StatusUnauthorized)
				return
			}
			sendError(writer, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}

		// todo шляпа - сторадж протек в контролер, мб сделать DI контейнер
		_, err = store.UpdateLastIncome(pseudonym, time.Now())

		handlerFunc(writer, request)
		return
	}
}

func sendError(writer http.ResponseWriter, errMsg string, httpStatus int) {
	writer.WriteHeader(httpStatus)
	json.NewEncoder(writer).Encode(struct {
		Error string `json:"error"`
	}{
		Error: errMsg,
	})
}
