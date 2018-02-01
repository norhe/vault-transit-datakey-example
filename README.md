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

Please note that the above command runs Vault in dev mode which means that secrets will not be persisted to disk.  If you stop the Vault process you will not be able to read records saved using any keys it created.  You will need to wipe the records from the database, and begin testing with new records.  

You then need to run the app:

```
VAULT_TOKEN=root go run main.go
```

You can then view the app using a browser at http://localhost:1234.

You can inspect the contents of the database with:
```
docker exec -it mysql-transit mysql -uroot -proot
```

Once you have added some records you can inspect them by opening the mysql client using the above command, and looking at records:

```
USE my_app;
SELECT * FROM user_data LIMIT 10;
# look at the data returned and observe the encryption
SELECT user_id, file_id, mime_type, file_name FROM user_files LIMIT 10;
# if you select the file contents itself say good bye to your terminal... 
```

NB:
Please note that this is my first foray into Go.  It is not well written or idiomatic.  The purpose of this is to demonstrate one possible way of encrypting both arbitrary text and larger amounts of data (like files) using Vault's transit feature.  Many improvements can be made including making things more Go-like, adding a check for file size as the transit engine can encrypt requests up to something like 32 MB, etc.  As time permits I would like to make these improvements, but if you stumble upon this before then consider yourself warned...
