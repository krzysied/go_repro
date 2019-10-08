good:
	./script_good.sh
	cat server.log | grep "Perc"
bad:
	./script_bad.sh
	cat server.log | grep "Perc"
clean:
	rm http_memory || true
	rm *.log || true
