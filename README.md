docker build -t blogagg:0.0.1 .

docker run --name blogAgg -e DATABASE_URL="postgres://myuser:mypassword@my-postgres/mydatabase?sslmode=disable" -d -p 8080:8080 blogagg:0.0.1
