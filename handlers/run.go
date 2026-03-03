package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func RunGoidaHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("./bin/goida_lang", "run", "./calculator.goida")

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Ошибка при выполнении goida_lang: %v", err)
		log.Printf("Вывод ошибки: %s", string(output))

		w.WriteHeader(http.StatusInternalServerError)
		if _, err = fmt.Fprintf(w, "Ошибка сервера (Exit Status: %v)\nПодробности: %s", err, string(output)); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Println("Успешное выполнение команды")
	if _, err := fmt.Fprintf(w, "Результат вычислений:\n%s", string(output)); err != nil {
		log.Fatal(err)
	}

}
