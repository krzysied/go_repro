package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type ClientPool struct {
	endpoint  string
	qps       float32
	certPath  string
	poolSize  int
	threshold time.Duration

	pool      []*http.Client
	lock      sync.Mutex
	durations []time.Duration
}

func NewClientPool(endpoint string, qps float32, certPath string, poolSize int, threshold time.Duration) (*ClientPool, error) {
	cp := &ClientPool{
		endpoint:  endpoint,
		qps:       qps,
		certPath:  certPath,
		poolSize:  poolSize,
		threshold: threshold,
	}
	if err := cp.init(); err != nil {
		return nil, err
	}
	return cp, nil
}

func (cp *ClientPool) Run(stopCh <-chan struct{}) {
	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Duration(int64(1.0 / cp.qps * (float32)(time.Second))))
	defer ticker.Stop()
	for i := 0; i < len(cp.pool); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					go cp.getRequest(i)
				}
			}
		}(i)
	}
	wg.Wait()
}

func (cp *ClientPool) getRequest(clientIndex int) {
	t := time.Now()
	if _, err := cp.pool[clientIndex].Get(cp.endpoint); err != nil {
		log.Printf("Client %d error: %v", clientIndex, err)
		return
	}
	d := time.Since(t)
	if cp.threshold != time.Duration(0) && d > cp.threshold {
		log.Printf("Client %d call duration %v", clientIndex, d)
	}
	cp.lock.Lock()
	defer cp.lock.Unlock()
	cp.durations = append(cp.durations, d)
}

func (cp *ClientPool) init() error {
	caCert, err := ioutil.ReadFile(cp.certPath)
	if err != nil {
		return fmt.Errorf("Reading server certificate error: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cp.pool = make([]*http.Client, cp.poolSize)
	for i := 0; i < len(cp.pool); i++ {
		cp.pool[i] = &http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	}
	return nil
}

func (cp *ClientPool) PrintMetric() {
	cp.lock.Lock()
	defer cp.lock.Unlock()
	if len(cp.durations) == 0 {
		log.Printf("Nothing to print")
		return
	}
	sort.Sort(durationAsc(cp.durations))
	log.Printf("Perc  50: %v", cp.durations[len(cp.durations)/2])
	log.Printf("Perc  90: %v", cp.durations[len(cp.durations)*9/10])
	log.Printf("Perc  99: %v", cp.durations[len(cp.durations)*99/100])
	log.Printf("Perc  99.9: %v", cp.durations[len(cp.durations)*999/1000])
	log.Printf("Perc  99.99: %v", cp.durations[len(cp.durations)*9999/10000])
	log.Printf("Perc 100: %v", cp.durations[len(cp.durations)-1])
}

type durationAsc []time.Duration

func (a durationAsc) Len() int           { return len(a) }
func (a durationAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a durationAsc) Less(i, j int) bool { return int(a[i]) < int(a[j]) }
