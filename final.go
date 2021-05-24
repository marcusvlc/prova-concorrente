package main

import (
	"time"
	"sync"
	"fmt"
)

type Request struct {
	size int
}

func getBucketLimit() int {
	return 10
}

func getRvalue() int {
	return 1
}

func startBucketIncrement(bucketCh chan int32, decrementBucketCh chan Request, doneWithRequestCh chan Request) {
	// Inicializa o canal de ticks para colocar um token no bucket a cada 1/R segundos
	tickCh := time.Tick(time.Duration(1 / getRvalue()) * time.Second)

	for {
		select {
		// A cada 1/r segundos um token é colocado no bucket
		case <-tickCh:
			bucketCh <- 1
		// Caso exista uma request a ser processada, entra nesse case.
		case req := <- decrementBucketCh:
			fmt.Printf("Qtd de tokens no bucket: %d // Request a ser processada (size): %d \n", len(bucketCh), req.size)
			// Se o bucket tiver tokens o suficiente pra processar a request, continua.
			if len(bucketCh) > req.size {
				// Retira N tokens do bucket onde N é o tamanho da request (size)
				for i:= 0; i < req.size; i++ {
					<- bucketCh
				}

				// Envia a request para o canal responsavel por avisar ao limitCap que a request foi processada.
				doneWithRequestCh <- req
				fmt.Printf("REQUEST PROCESSADA! DORMINDO APENAS PARA FINS DE DEBUG... \n")
				time.Sleep(time.Duration(1)  * time.Second)
			// Caso contrario, re-coloca a request no canal responsavel por processar as requests.
			// Esse passo é necessário visto que, no case, a request é retirada de lá e pode existir o cenário
			// onde o bucket não tem tokens suficientes, então ela deve voltar mais tarde.
			} else {
				decrementBucketCh <- req
			}

		}
	}
}

func limitCap_wait(req Request, wg *sync.WaitGroup, decrementBucketCh chan Request, doneWithRequestCh chan Request) {
	// Se a requisicao tiver o tamanho maior que o B, é retornado e nada é feito.
	if(req.size > getBucketLimit()) {
		defer wg.Done()
		return
	}
	// Envia a request para o canal responsável por processar as requests caso haja espaço no bucket.
	go func() {
		decrementBucketCh <- req
	}()
	
	select {
	// Case que ouve o canal de finalização para terminar a execução da goroutine
	case reqFromCh := <- doneWithRequestCh:
		// Caso a request que apareceu no canal doneWithRequestCh seja a request dessa goroutine, siginfica que ela foi processada
		// Portanto, a execucao finaliza.
		if req == reqFromCh {
			defer wg.Done()
			return
		// Se nao for a request dessa goroutine, re-coloca no canal.
		} else {
			doneWithRequestCh <- reqFromCh
		}
	}
	

}

func run(req Request, wg *sync.WaitGroup, decrementBucketCh chan Request, doneWithRequestCh chan Request) {
	limitCap_wait(req, wg, decrementBucketCh, doneWithRequestCh)
}

func main() {
	// Numero de requests que serao simuladas (chamadas a funcao run)
	n_requests := 3
	// Inicia os 3 canais utilizados na solucao.
	// O bucketCh sera o canal resposavel por guardar os tokens.
	// O decrementBucketCh sera o canal responsavel por armazenar as requests a serem processadas
	// O donewithRequestCh sera o canal responsavel por armazenar as requests já processadas
	bucketCh := make(chan int32, getBucketLimit())
	decrementBucketCh := make(chan Request, n_requests)
	doneWithRequestCh := make(chan Request, n_requests)
	// Inicializa uma go routine para ficar incrementando o bucket
	go startBucketIncrement(bucketCh, decrementBucketCh, doneWithRequestCh)

	var wg sync.WaitGroup

	// Loop que simula requisicoes criando goroutines da funcao run
	// Por padrao, cria 2 requests de size 5.
	for i:= 0; i<n_requests; i++ {
		wg.Add(1)
		size := 3
		request := Request { size:  int(size)}
		go run(request, &wg, decrementBucketCh, doneWithRequestCh)
	}

	wg.Wait()
}
