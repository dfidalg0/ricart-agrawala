# Algoritmo de Ricart-Agrawala

Este repositório contém uma implementação em Golang do [algoritmo de Ricart-Agrawala](https://en.wikipedia.org/wiki/Ricart%E2%80%93Agrawala_algorithm), utilizado para controle de acesso a recursos críticos em sistemas distribuídos.

**Autor**: Diego Teixeira Nogueira Fidalgo

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
make process # Compila o binário do processo

make shared # Compila o binário da região crítica
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

A região crítica é um simples servidor UDP que aceita mensagens no formato `cs<clock,id>(text)`, onde `clock` é o relógio lógico do processo que faz a requisição, `id` o seu id e `text` um texto arbitrário, e as imprime na tela com a formatação adequada.

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

Nesta linha de execução, encontram-se as variáveis de estado

* `clock` - Relógio lógico atual do processo, iniciado em 1
* `lastRequestClock` - Relógio lógico da última vez que o processo solicitou a região crítica
* `state` - Estado de uso da região crítica pelo processo (pode ser FREE, WAIT ou HELD)
* `replyQueue` - Fila de funções de resposta

As quais são modificadas exclusivamente nesta mesma linha durante o processamento do loop principal.

#### Mensagens

O envelope de cada mensagem é uma `struct` com os campos `Source`, `Text` e `Reply`. Estes são, respectivamente, a origem da mensagem (teclado, UDP ou região crítica), o texto da mensagem e uma função que, quando chamada com o clock do processo atual e o seu id, responde à mensagem.

As mensagens trocadas passadas pelo teclado são simplesmente o texto obtido da entrada padrão sem uma função de resposta e as trocadas por meio do UDP vindas de outros processos têm o formato `req<clock,id>` com uma função de resposta que, quando chamada, envia uma mensagem `reply<clock,id>` através da mesma conexão. Em ambos os casos, `clock` e `id` representam o relógio lógico e o id do processo remetente da mensagem.

#### Acesso à região crítica

O acesso à região crítica é feito por meio de uma função, executada em uma *goroutine*, que requisita todos os demais processos e espera uma resposta. Esta função recebe como argumento um callback que permite o acesso à região crítica.

Nesta implementação, o único callback utilizado envia uma mensagem para a região crítica ao adentrá-la, espera 5 segundos e envia uma nova mensagem antes de sair.

## Casos de Teste

<div style="display: flex; width: 100%; flex-wrap: wrap; box-sizing: border-box;">
    <div style="flex-basis: 65%; flex-grow: 1;">
        Os casos de teste abaixo consistem de algumas situações hipotéticas de execução do algoritmo e suas realizações. Cada caso consiste de uma descrição verbal da situação, um diagrama lógico do comportamento esperado do algoritmo e uma demonstração de sua execução. Com isso, visa-se comprovar o funcionamento do algoritmo nestas situações.
        <br><br>
        Nos diagramas lógicos, a simbologia utilizada corresponde à legenda apresentada nesta seção.
    </div>
    <div style="flex-basis: 35%; flex-grow: 1; display: flex; justify-content: flex-end;">
        <img
            style="width: 300pt;"
            src="https://i.imgur.com/kUhqvga.png"
        ></img>
    </div>
</div>

### Caso 1

> Um processo solicita a região crítica e tem o acesso garantido. Quando este processo termina de usar a região crítica, outro processo solicita o acesso.

Para este caso, é esperado o seguinte comportamento dos processos.

![Diagrama 1](https://i.imgur.com/WkXGocm.png)

O que corresponde ao resultado observado na execução abaixo

![Caso 1](https://i.imgur.com/0YKoa3f.png)

Aqui, o processo 1 tem o seu acesso à região crítica sem problemas e o processo 2 igualmente. Não há nenhum tipo de conflito por uso da região crítica e todos os replies são imediatos. Este teste serve para mostrar que o algoritmo funciona no seu caso mais básico, garantindo que todos os processos conseguem acesso à região crítica sem sobreposição.

### Caso 2

> Um processo solicita a região crítica e tem o acesso garantido. Enquanto este processo ainda está fazendo uso da região crítica, outro processo faz a solicitação.

Para este caso, é esperado o seguinte comportamento dos processos.

![Diagrama 2](https://i.imgur.com/fVWYNiQ.png)

O que corresponde ao resultado observado

![Caso 2](https://i.imgur.com/026Gu4K.png)

Neste caso, o processo 2 requisita a seção crítica antes do processo 1 terminar o uso desta. Assim, o processo 3 responde o processo 2 antes do processo 1, de forma que o relógio lógico do processo 2 passa a ser 6 e não 7 como no caso anterior.

Este teste, todavia, mostra que o algoritmo funciona também no caso de duas requisições simultâneas à seção crítica, garantindo que apenas um processo a acessa por vez e que todos os processos que a solicitam eventualmente conseguem utilizá-la. Isso ocorre em razão do processo 1 adiar a resposta ao processo 2 para o momento em que termina de utilizar a seção crítica.

### Caso 3

> Um processo solicita a região crítica e tem o acesso garantido. Enquanto este processo ainda está fazendo uso da região crítica, outro processo faz a solicitação. Antes do segundo processo ter seu acesso concedido, um terceiro processo solicita a região crítica.

Para este caso, é esperado o seguinte comportamento dos processos.

![Diagrama 3](https://i.imgur.com/BuxL2fP.png)

O que corresponde ao resultado observado

![Caso 3](https://i.imgur.com/cmUDPwj.png)

Neste caso, o processo 3 têm de esperar dois processos terminarem a utilização da região crítica para que possa fazer o uso desta. Isso ocorre pois, quando o processo 3 envia a sua solicitação, o processo 1 adia a resposta para o momento em que termina de utilizar a seção crítica, por estar utilizando-a e o processo 2 se comporta da mesma forma, no entanto, em razão de estar esperando ter seu acesso garantido e ter feito o pedido antes do processo 3.

Este é um caso mais complexo na execução do algoritmo, que recorre ao uso do relógio lógico dos processos para garantir que todos os processos conseguem eventualmente acessar a região crítica e não há sobreposição entre os acessos.

Esta recorrência ocorre no momento que o processo 2 recebe a solicitação do processo 3, caso em que o estado de 2 é de espera por acesso à região crítica e o fator decisivo para o adiamento da resposta a esta solicitação é o resultado da comparação dos relógios lógicos de 2 e 3.

O teste, portanto, prova o funcionamento do algoritmo para o este caso.
