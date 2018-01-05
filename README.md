# Vault Transit Datakey Example


## Usage

You need a database to test with:

```
docker pull mysql/mysql-server:5.7
mkdir ~/transit-data
docker run --name mysql-transit \
  -p 3306:3306 \
  -v ~/transit-data:/var/lib/mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_ROOT_HOST=% \
  -e MYSQL_DATABASE=my_app \
  -e MYSQL_USER=vault \
  -e MYSQL_PASSWORD=vaultpw \
  -d mysql/mysql-server:5.7
```

To configure Vault:

```
vault server -dev -dev-root-token-id=root &
export VAULT_ADDR='http://127.0.0.1:8200'
vault mount transit
vault write -f transit/keys/my_app_key
```

You then need to run the app:

```
go run main.go
```

You can then view the app using a browser at http://localhost:1234.

You can inspect the contents of the database with:
```
docker exec -it mysql-transit mysql -uroot -proot
```
