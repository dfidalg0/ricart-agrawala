package message

// Envelope para unificação das mensagens que chegam ao processo
type Message struct {
	Source MessageSource
	Text   string
	Reply  func()
}

// Enumeração das possíveis fontes de mensagens que chegam ao processo
type MessageSource int

const (
	KEYBOARD MessageSource = 0
	UDP      MessageSource = 1
	CS       MessageSource = 2
)
