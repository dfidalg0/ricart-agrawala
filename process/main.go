package main

import (
	"fmt"
	"os"
	"process/cs"
	"process/keyboard"
	"process/message"
	"process/server"
	"process/states"
	"strconv"
	"time"
)

func main() {
	// Verificação inicial de uso correto da linha de comando
	nArgs := len(os.Args)

	if nArgs < 3 {
		fmt.Println("Usage:", os.Args[0], "[id] [...ports]")
		os.Exit(1)
	}

	// Captura de parâmetros
	idStr := os.Args[1]

	pid, err := strconv.Atoi(idStr)

	// Obtenção do número de processos
	nProcesses := nArgs - 2

	// Validação do id do processo
	if err != nil || pid < 1 || pid > nProcesses {
		fmt.Println("ID inválido:", idStr)
		os.Exit(1)
	}

	// Porta de escuta do processo atual
	port := os.Args[pid+1]

	// Esse canal é criado para centralizar a chegada de mensagens de todas as
	// fontes em um único local
	channel := make(chan message.Message, 10)

	// Cada fonte executa o seu trabalho individualmente e retransmite
	// para o canal uma mensagem envelopada. Este, por fim, sincroniza
	// as mensagens, retirando a necessidade de implementação de um Mutex
	go server.Listen(port, channel)
	go keyboard.Listen(channel)

	// Inicialização de variáveis de estado
	clock := 0
	state := states.FREE
	lastRequestClock := -1

	// Inicialização da fila de replies
	var replyQueue []func()

	// Loop de eventos de leitura do canal, feita de forma síncrona
	for msg := range channel {
		// Verificamos, inicialmente, a fonte da mensagem

		if msg.Source == message.KEYBOARD { // Caso 1: Input do teclado
			// Se o texto digitado for "x" e o estado atual do processo for FREE,
			// um pedido de acesso à CS deve ser feito
			typedX := msg.Text == "x"
			requestCS := typedX && state == states.FREE

			// Além disso, o relógio lógico deve ser incrementado. Isso também
			// deve ocorrer quando o texto é igual ao id do processo
			updateClock := requestCS || msg.Text == idStr

			if updateClock {
				// Atualização do relógio lógico
				clock += 1
			}

			// O código à frente é destinado a requisitar a região crítica e usá-la
			if !typedX {
				continue
			}

			// Uma requisição à CS só será feita se o processo estiver livre
			if !requestCS {
				fmt.Println("'x' ignored")
				continue
			}

			// Atualização do clock da última requisição à CS
			lastRequestClock = clock
			// Atualização do estado
			state = states.WAIT

			// Criação de uma goroutine para requisitar a sessão crítica
			// sem bloquear o loop de leitura do canal
			go cs.Request(pid, clock, func(send cs.Sender) {
				// Quando o acesso à região crítica é finalmente concedido
				// o processo deve imediatamente se colocar como "HELD"
				state = states.HELD

				// E assim interagir com a região crítica
				msg := fmt.Sprintf("cs<%d,%d>(interaction with cs begins)", clock, pid)
				send(msg)

				// Tempo de espera para simular um atraso na utilização da CS
				time.Sleep(time.Second * 5)

				msg = fmt.Sprintf("cs<%d,%d>(interaction with cs ends)", clock, pid)
				send(msg)

				// Assim que a CS termina de ser utilizada, o estado deve
				// ser atualizado para FREE
				state = states.FREE

				// E uma mensagem deve ser enviada ao canal de sincronização
				// para evitar problemas de coerência na variável "replyQueue"
				channel <- message.Message{
					Source: message.CS,
					Text:   "end",
					Reply: func() {
						if len(replyQueue) == 0 {
							return
						}

						// E os replies agendados devem ser executados
						for _, Reply := range replyQueue {
							Reply()
						}

						// E, finalmente, a fila de replies deve ser limpa
						replyQueue = make([]func(), 0)
					},
				}
			})
		} else if msg.Source == message.UDP { // Caso 2: Requisição UDP
			// Primeiro gravamos os dados da requisição
			var reqClock int
			var reqPid int

			fmt.Sscanf(msg.Text, "req<%d,%d>", &reqClock, &reqPid)

			// Primeiro devemos atualizar o clock do processo

			// Caso o clock da requisição seja superior, devemos trocar o clock
			// original por este
			if reqClock > clock {
				clock = reqClock
			}

			// E, independentemente do resultado, incrementar o clock do processo
			clock += 1

			// Verificamos se o processo que mandou a requisição tem prioridade
			// sobre a CS
			reqHasPriority := reqClock < lastRequestClock || (reqClock == lastRequestClock && reqPid < pid)

			// E, caso tenha, ou caso o processo corrente esteja livre,
			// este processo recebe um reply imediato
			if state == states.FREE ||
				(state == states.WAIT && reqHasPriority) {
				msg.Reply()
			} else {
				// Caso contrário, o reply é agendado na fila
				replyQueue = append(replyQueue, msg.Reply)
			}
		} else if msg.Source == message.CS { // Caso 3: Mensagem da seção crítica
			msg.Reply()
		} else { // Caso 4: Nunca vai acontecer, mas vai que...
			fmt.Println("Invalid message source")
			os.Exit(1)
		}
	}
}
