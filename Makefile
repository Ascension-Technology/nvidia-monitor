all:
	docker build -t nvidia-monitor .
	docker run --env-file=.env -it nvidia-monitor

clean:
	rm -rf tmp/