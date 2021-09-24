package keyboard

import (
	"bufio"
	"os"
	"process/message"
)

// Ouve eventos do teclado e os retransmite para o canal passado
func Listen(channel chan message.Message) {
	// Leitor do teclado
	reader := bufio.NewReader(os.Stdin)

	for {
		// Leitura de uma linha do teclado
		text, _, err := reader.ReadLine()

		// Em caso de erro, significa que esse input está desabilitado
		if err != nil {
			break
		}

		// Retransmissão da mensagem para o canal
		channel <- message.Message{
			Source: message.KEYBOARD,
			Text:   string(text),
			Reply:  nil,
		}
	}
}
