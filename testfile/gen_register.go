package main

import (
	"bufio"
	"os"

	"github.com/google/uuid"
)

const passwd = "123"

func main() {
	filename := "notRegister.txt"
	file, _ := os.Create(filename)
	writer := bufio.NewWriter(file)
	for i := 0; i < 1000; i++ {
		usr, _ := uuid.NewRandom()
		writer.WriteString(usr.String()[0:8] + " " + passwd + "\n")
	}
	writer.Flush()

}
