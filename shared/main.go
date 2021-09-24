package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Resolução do endereço UDP
	// Aqui, a porta utilizada é a 10000
	addr, err := net.ResolveUDPAddr("udp", ":10000")

	checkError(err)

	// Criação do socket UDP
	conn, err := net.ListenUDP("udp", addr)

	checkError(err)

	// Fechamento da conexão ao fim da execução
	defer conn.Close()

	// Criação do buffer de leitura
	buf := make([]byte, 4096)

	for {
		// Leitura do socket
		n, _, err := conn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("Error:", err)
		} else {
			// Recuperação da mensagem
			recv := string(buf[0:n])

			// Formato da mensagem: "cs<clock,pid>(text)"

			// Índice de início do texto
			begin := strings.Index(recv, "(") + 1
			// Índice de fim do texto
			end := len(recv) - 1

			// Variáveis auxiliares
			var pid int
			var clock int
			fmt.Sscanf(recv, "cs<%d,%d>", &clock, &pid)

			// Obtenção do texto
			text := recv[begin:end]

			// Impressão da mensagem no terminal
			fmt.Printf("[PID: %d, Clock: %d] %s\n", pid, clock, text)
		}
	}
}

func checkError(err error) {
	if err != nil {
		os.Exit(1)
	}
}
