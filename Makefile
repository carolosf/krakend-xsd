# Need to put some of these in docker compose for easier testing
quick_build_docker:
	docker buildx build --platform linux/amd64 -f Dockerfile.quickbuild -t quickbuild-krakend-xsd .

build_runner:
	docker buildx build --platform linux/amd64 -f Dockerfile.krakendlibxml2 -t krakend-libxml2 .

quick_build:
	docker run --rm --name krakend-xsd-quick-build --rm -it -v "${PWD}:/app" --platform linux/amd64 -e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 -w /app quickbuild-krakend-xsd sh -c "go build -buildmode=plugin -o plugins/krakend-xsd.so ."

quick_run:
	docker run --rm --name krakend-xsd-quick-run --platform linux/amd64 -p "8080:8080" -v "${PWD}:/etc/krakend/" -v "${PWD}/plugins:/opt/krakend/plugins/" -v "${PWD}/xsd:/opt/krakend/xsd/" krakend-libxml2 run -c /etc/krakend/krakend.json

quick_run_response:
	docker run --rm --name krakend-xsd-quick-run-response --platform linux/amd64 -p "8083:8080" -v "${PWD}:/etc/krakend/" -v "${PWD}/plugins:/opt/krakend/plugins/" -v "${PWD}/xsd:/opt/krakend/xsd/" krakend-libxml2 run -c /etc/krakend/krakend2.json

quick_build_and_run: quick_build quick_run

run_nginx:
	docker run --rm --name test-nginx -d -p 8081:80 nginx

run_nginx_response:
	docker run --rm --name test-nginx-response -p 8082:80 -v "${PWD}/docker-nginx-response/static:/usr/share/nginx/html:ro" -d nginx

curl_post_body_valid:
	curl -X POST -H "Content-Type: application/xml" -H "Accept: application/xml" --data-binary "@docker-nginx-response/static/valid.xml" --trace curl.log -vvv http://localhost:8080/

curl_post_body_invalid:
	curl -X POST -H "Content-Type: application/xml" -H "Accept: application/xml" --data-binary "@docker-nginx-response/static/invalid.xml" --trace curl.log -vvv http://localhost:8080/
