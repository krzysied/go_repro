package main

import (
	"flag"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	typeFlag          = flag.String("type", "", "component type")
	endpointFlag      = flag.String("endpoint", "", "endpoint for client request")
	qpsFlag           = flag.Float64("qps", 1, "clientQps")
	certPathFlag      = flag.String("certPath", "", "cert path")
	poolSizeFlag      = flag.Int("poolSize", 1, "client pool size")
	thresholdFlag     = flag.Duration("threshold", time.Duration(0), "client call threshold")
	portFlag          = flag.Int("port", 8080, "server port")
	keyPathFlag       = flag.String("keyPath", "", "key path")
	sleepDurationFlag = flag.Duration("sleepDuration", 50*time.Millisecond, "sleep duration for calls handler")
	durationFlag      = flag.Duration("duration", 20*time.Minute, "component run duration")
)

func main() {
	flag.Parse()
	var comp RunnableMetrics
	switch *typeFlag {
	case "client":
		comp = createClient(*endpointFlag, (float32)(*qpsFlag), *certPathFlag, *poolSizeFlag, *thresholdFlag)
	case "apiserver":
		comp = createApiserver(*endpointFlag, (float32)(*qpsFlag), *certPathFlag, *poolSizeFlag, *thresholdFlag, *portFlag, *keyPathFlag, *sleepDurationFlag)
	case "etcd":
		comp = createEtcd(*certPathFlag, *portFlag, *keyPathFlag, *sleepDurationFlag)
	default:
		log.Fatal("Unknown component type")
	}
	stopCh := make(chan struct{})
	time.AfterFunc(*durationFlag, func() {
		close(stopCh)
	})

	comp.Run(stopCh)
	comp.PrintMetric()
}

type RunnableMetrics interface {
	Run(stopCh <-chan struct{})
	PrintMetric()
}

func createClient(endpoint string, qps float32, certPath string, poolSize int, threshold time.Duration) RunnableMetrics {
	cp, err := NewClientPool(endpoint, qps, certPath, poolSize, threshold)
	if err != nil {
		log.Fatalf("Client creation error: %v", err)
	}
	return cp
}

type apiserver struct {
	server     RunnableMetrics
	clientPool RunnableMetrics
}

func createApiserver(endpoint string, qps float32, certPath string, poolSize int, threshold time.Duration,
	port int, keyPath string, sleepDuration time.Duration) RunnableMetrics {
	clientPool, err := NewClientPool(endpoint, qps, certPath, poolSize, threshold)
	if err != nil {
		log.Fatalf("Apiserver creation error: %v", err)
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Log the request protocol
		tab := make([][]byte, 80)
		for i := 0; i < len(tab); i++ {
			tab[i] = make([]byte, 80000/len(tab))
		}
		log.Printf("Got connection: sleep: %s", r.Proto)
		time.Sleep(sleepDuration)
		log.Printf("Close connection: sleep: %s", r.Proto)
		w.WriteHeader(http.StatusOK)
		for i := 0; i < len(tab); i++ {
			tab[i][0] = 1
		}
	}
	server := NewServer(port, certPath, keyPath, handler)

	return &apiserver{
		clientPool: clientPool,
		server:     server,
	}
}

func (as *apiserver) Run(stopCh <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		as.server.Run(stopCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		as.clientPool.Run(stopCh)
	}()
	wg.Wait()
}

func (as *apiserver) PrintMetric() {
	as.clientPool.PrintMetric()
}

func createEtcd(certPath string, port int, keyPath string, sleepDuration time.Duration) RunnableMetrics {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Log the request protocol
		log.Printf("Got connection: %s", r.Proto)
		time.Sleep(sleepDuration)
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, 80000))
	}

	return NewServer(port, certPath, keyPath, handler)
}
