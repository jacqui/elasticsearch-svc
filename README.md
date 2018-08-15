# elasticsearch-svc

webservice written in golang, built with docker-compose, that can post to and search documents in elasticsearch

# setup

```
cd $GOPATH/github.com/jacqui
git clone git@github.com:jacqui/elasticsearch-svc.git
cd elasticsearch-svc
docker-compose up -d --build
curl -X POST http://localhost:8080/documents -d @fake-data.json -H "Content-Type: application/json"
curl http://localhost:8080/search?query=exercitation+est+officia
```

