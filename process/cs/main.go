package cs

import (
	"fmt"
	"net"
	"os"
	"process/message"
	"strconv"
)

// Cache das conexões UDP
var csConnection *net.UDPConn = nil
var connections []*net.UDPConn

// Número total de processos executando o algoritmo
var nProcesses int

// Função de contato com a região crítica
type Sender func(string)

// Requisita acesso à região crítica
// Para tal, é necessário que sejam fornecidos o id do procesos, o seu relógio
// lógico e uma função de callback que será executada quando o acesso for
// garantido
func Request(pid int, clock int, channel chan message.Message, onHeld func(send Sender)) {
	// Primeiro, devemos inicializar as conexões com os demais processos
	// e com a região crítica
	fillConnections(pid)

	// Criamos um canal para sincronizar as respostas dos outros processos
	auxChannel := make(chan bool, nProcesses-1)

	// E os requisitamos um a um
	for _, conn := range connections {
		// Em paralelo, utilizando uma goroutine para cada requisição
		go func(conn *net.UDPConn) {
			// Formatamos a requisição
			msg := "req<" + strconv.Itoa(clock) + "," + strconv.Itoa(pid) + ">"

			// Enviamos-a a seu destino
			conn.Write([]byte(msg))

			// Aguardamos uma resposta
			buf := make([]byte, 100)

			n, _ := conn.Read(buf)

			// E a encaminhamos para o canal de sincronização
			channel <- message.Message{
				Source: message.UDP,
				Text:   string(buf[0:n]),
				Reply:  nil,
			}

			// E por fim, atualizamos o canal para indicar que um dos processos
			// permitiu o acesso à região crítica
			auxChannel <- true
		}(conn)
	}

	// Esperamos então até que todos os processos tenham concedido acesso à CS
	for i := 0; i < nProcesses-1; i += 1 {
		<-auxChannel
	}

	// E fechamos o canal
	close(auxChannel)

	// Como o acesso à região crítica é feito através de um callback,
	// usamos essa flag para garantir que não haverão problemas
	access := true

	// Definimos então a função de comunicação do processo com a região crítica
	send := func(msg string) {
		if !access {
			panic("Unallowed access to critical section")
		}

		csConnection.Write([]byte(msg))
	}

	// E então chamamos a função de callback passando "send" como argumento
	onHeld(send)

	// Por fim, desabilitamos a flag de acesso, garantindo que não haverão
	// acessos indevidos à região crítica
	access = false
}

// Cria todas as conexões necessárias caso elas ainda não existam
func fillConnections(pid int) {
	if csConnection == nil {
		localAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		ports := os.Args[2:]
		nProcesses = len(ports)
		for i, port := range ports {
			if i+1 == pid {
				continue
			}

			addr, err := net.ResolveUDPAddr("udp", port)

			if err != nil {
				fmt.Println("Process", i+1, "not connected")
			}

			conn, err := net.DialUDP("udp", localAddr, addr)

			if err != nil {
				fmt.Println("Process", i+1, "not connected")
			}

			connections = append(connections, conn)
		}

		csAddr, _ := net.ResolveUDPAddr("udp", ":10000")

		conn, err := net.DialUDP("udp", localAddr, csAddr)
		csConnection = conn

		if err != nil {
			fmt.Println("Critical section disconnected")
			os.Exit(1)
		}
	}
}
