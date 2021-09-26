# Algoritmo de Ricart-Agrawala

Este repositório contém uma implementação em Golang do [algoritmo de Ricart-Agrawala](https://en.wikipedia.org/wiki/Ricart%E2%80%93Agrawala_algorithm), utilizado para controle de acesso a recursos críticos em sistemas distribuídos.

## Modularização

A estrutura do código fonte está organizada nos módulos `process` e `shared`. O primeiro corresponde a um dos processos que farão a utilização de uma seção crítica, cujo código fonte está no módulo `shared`.

O módulo `process` possui os submódulos `cs`, `keyboard`, `message`, `server` e `states`, utilizados para organização interna do código fonte.

Quando compilados, os módulos geram os executáveis `bin/process` e `bin/shared`.

## Compilação

Todos os executáveis podem ser compilados através do comando

```bash
make
```

Ou, individualmente, através dos comandos

```bash
make process

make shared
```

## Utilização

Após compilados os executáveis, o servidor da região crítica pode ser aberto com o comando

```bash
./bin/shared
```

E os servidores de cada processo podem ser abertos com o comando

```bash
./bin/process [id] [...ports]
```

Onde `id` é o id do processo e ```ports``` é uma lista de portas no formato `:PORT` separadas por espaços correspondente às portas utilizadas por cada processo no algoritmo.

**Atenção**: Não usar a porta `:10000`, pois esta está reservada para o servidor da seção crítica.

### Processos

Cada processo aberto usa um terminal para capturar eventos do teclado e uma conexão UDP para receber mensagens de outros processos. Ao receber a mensagem `x` no terminal, um processo irá requisitar aos outros o uso da seção crítica e, ao receber o seu próprio id, irá incrementar seu relógio lógico. Todos os demais inputs são ignorados.

## Implementação

### Região crítica

A região crítica é um simples servidor UDP que aceita mensagens no formato `cs<clock,id>(text)` e as imprime na tela com a formatação adequada.

### Processo

Cada processo é implementado como uma linha principal de execução que processa mensagens de diversas fontes. Essas fontes são o servidor UDP interno do processo, o terminal do processo e a thread da região crítica.

Cada uma dessas fontes escreve em um canal unificado uma mensagem envelopada e a linha de execução principal apenas consome este canal e processa as mensagens, de forma a sincronizar todas as origens e garantir consistência de estado ao processo como um todo.

Dessa forma, a linha de execução principal pode ser resumida como

```go
for msg := range channel {
    if (msg.Source == KEYBOARD) {
        // Process keyboard message
    } else if (msg.Source == UDP) {
        // Process UDP message
    } else if (msg.Source == CS) {
        // Process CS message
    }
}
```

#### Mensagens

O envelope de cada mensagem é uma `struct` com os campos `Source`, `Text` e `Reply`. Estes são, respectivamente, a origem da mensagem (teclado, UDP ou região crítica), o texto da mensagem e uma função sem argumentos que, quando chamada, responde à mensagem.

As mensagens trocadas passadas pelo teclado são simplesmente o texto obtido da entrada padrão sem uma função de resposta e as trocadas por meio do UDP vindas de outros processos têm o formato `req<clock,id>` com uma função de resposta que, quando chamada, envia uma mensagem `reply<req<clock,id>>` através da mesma conexão.

#### Acesso à região crítica

O acesso à região crítica é feito por meio de uma função, executada em uma *goroutine*, que requisita todos os demais processos e espera uma resposta. Esta função recebe como argumento um callback que permite o acesso à região crítica.

Nesta implementação, o único callback utilizado envia uma mensagem para a região crítica ao adentrá-la, espera 5 segundos e envia uma nova mensagem antes de sair.

## Testes

Em breve.
