package server

import (
	"fmt"
	"net"
	"os"
	"process/message"
)

// Ouve as requisições UDP na porta espeficada e as retransmite pelo canal
// passado
func Listen(port string, channel chan<- message.Message) {
	// Resolução do endereço
	addr, err := net.ResolveUDPAddr("udp", port)

	checkError(err)

	// Criação do socket de leitura UDP
	conn, err := net.ListenUDP("udp", addr)

	checkError(err)

	// Fechamento da conexão ao fim da função
	defer conn.Close()

	// Criação do buffer de leitura
	buf := make([]byte, 4096)

	for {
		// Leitura da conexão UDP
		n, addr, err := conn.ReadFromUDP(buf)

		if err != nil {
			fmt.Println("Error:", err)
			break
		}

		// Obtenção do texto da mensagem
		Text := string(buf[0:n])

		// Criação da função de resposta à requisição
		Reply := func(clock int, pid int) {
			msg := fmt.Sprintf("reply<%d,%d>", clock, pid)
			go conn.WriteTo([]byte(msg), addr)
		}

		// Retransmissão da mensagem pelo canal
		channel <- message.Message{
			Source: message.UDP,
			Text:   Text,
			Reply:  Reply,
		}
	}
}

func checkError(err error) {
	if err != nil {
		os.Exit(1)
	}
}
