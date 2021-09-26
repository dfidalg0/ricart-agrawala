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
	var replyQueue []func(int, int)

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

			// Criamos uma goroutine para requisitar a sessão crítica
			// sem bloquear o loop de leitura do canal. Passamos a esta rotina
			// uma função que será executada (pela própria rotina) quando o
			// processo conseguir acesso à região crítica
			go cs.Request(pid, clock, channel, func(send cs.Sender) {
				// Quando o processo adquire o acesso, devemos primeiro
				// atualizar o seu estado
				// Para isso, faremos uma comunicação com a thread principal
				// através do canal de sinconização

				// No entanto, na thread da região crítica,
				// devemos ter controle sobre a conclusão do processo
				// por isso criamos um segundo canal auxiliar
				done := make(chan bool)

				// E então enviamos a mensagem com todas as ações síncronas
				// que devem ser realizadas antes de utilizarmos, de fato, a CS
				channel <- message.Message{
					Source: message.CS,
					Text:   "enter",
					Reply: func(int, int) {
						// Quando o acesso à região crítica é finalmente concedido
						// o processo deve imediatamente se colocar como "HELD"
						state = states.HELD

						// Indicamos então que o processo acessou a CS
						fmt.Println("Process entered CS")

						// E informamos a thread da CS deste fato
						done <- true
					},
				}

				// Na thread da CS, aguardamos a conclusão do processo síncrono
				<-done

				// E então fechamos o canal auxiliar
				close(done)

				// Para assim, finalmente interagir com a região crítica de fato
				msg := fmt.Sprintf("cs<%d,%d>(interaction with cs begins)", clock, pid)
				send(msg)

				time.Sleep(time.Second * 5)

				msg = fmt.Sprintf("cs<%d,%d>(interaction with cs ends)", clock, pid)
				send(msg)

				// Logo em seguida, enviamos uma nova mensagem à thread principal
				// para realizar ações síncronas de conclusão do uso da CS
				channel <- message.Message{
					Source: message.CS,
					Text:   "leave",
					Reply: func(clock int, pid int) {
						// Assim que a CS termina de ser utilizada, o estado deve
						// ser atualizado para FREE
						state = states.FREE

						fmt.Println("Process exited CS")

						// Então verificamos se há algum reply pendente
						// caso não haja, podemos encerrar aqui
						if len(replyQueue) == 0 {
							return
						}

						// Caso haja algum, devemos executá-los
						for _, Reply := range replyQueue {
							Reply(clock, pid)
						}

						// E, finalmente, limpar a fila de replies
						replyQueue = make([]func(int, int), 0)
					},
				}
			})
		} else if msg.Source == message.UDP { // Caso 2: Tráfego UDP
			// Primeiro gravamos os dados do remetente
			var senderClock int
			var senderPid int

			// E verificamos o tipo de tráfego (requisição ou resposta)
			isRequest := msg.Text[0:3] == "req"

			isReply := msg.Text[0:5] == "reply"

			// Para, apropriadamente, capturar os parâmetros da mensagem
			if isRequest {
				fmt.Sscanf(msg.Text, "req<%d,%d>", &senderClock, &senderPid)
			} else if isReply {
				fmt.Sscanf(msg.Text, "reply<%d,%d>", &senderClock, &senderPid)
			}

			// Caso o clock do remetente seja superior, devemos trocar o clock
			// original por este
			if senderClock > clock {
				clock = senderClock
			}

			// E, independentemente do resultado, incrementar o clock do processo
			clock += 1

			// Caso a mensagem recebida não seja de uma requisição, continuamos
			// a leitura do canal para processar a próxima mensagem
			if !isRequest {
				continue
			}

			// Verificamos se o processo que mandou a requisição tem prioridade
			// sobre a CS
			reqHasPriority := senderClock < lastRequestClock || (senderClock == lastRequestClock && senderPid < pid)

			// E, caso tenha, ou caso o processo corrente esteja livre,
			// este processo recebe um reply imediato
			if state == states.FREE ||
				(state == states.WAIT && reqHasPriority) {
				msg.Reply(clock, pid)
			} else {
				// Caso contrário, o reply é agendado na fila
				replyQueue = append(replyQueue, msg.Reply)
			}
		} else if msg.Source == message.CS { // Caso 3: Mensagem da seção crítica
			// Nesse caso, é necessário apenas invocar o método da mensagem
			msg.Reply(clock, pid)
		} else { // Caso 4: Nunca vai acontecer, mas vai que...
			fmt.Println("Invalid message source")
			os.Exit(1)
		}
	}
}
