package secure

import (
	"github.com/hashicorp/vault/api"
	"log"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"encoding/base64"
	"crypto/rand"
)

var vlt *api.Client
const KEY_NAME = "my_app_key"

func init() {
	cfg := api.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8200"

	c, err := api.NewClient(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	vlt = c
}

func GetDatakey() (*api.Secret, error) {
	datakey, err := vlt.Logical().Write("transit/datakey/plaintext/" + KEY_NAME, nil)
	return datakey, err
}

func EncryptFile(contents []byte, key []byte) ([]byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("Error creating cipher: %s", err)
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Fatalf("Error creating nonce: %s", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("Error creating aesgcm: %s", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, contents, nil)
	c_text_w_nonce := make([]byte, cap(ciphertext) + 12)
	copy(c_text_w_nonce[0:12], nonce)
	copy(c_text_w_nonce[12:], ciphertext)

	return c_text_w_nonce
}

func DecryptFile(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("Error creating cipher: %s", err)
	}

	nonce := ciphertext[0:12]

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("Error creating aesgcm: %s", err)
	}

	contents, err := aesgcm.Open(nil, nonce, ciphertext[12:], nil)
	if err != nil {
		log.Fatalf("Error decrypting file: %s", err)
	}
	return contents, err
}

func DecryptString(ciphertext string) ([]byte, error) {
	decrypted_contents, err := vlt.Logical().Write("transit/decrypt/" + KEY_NAME, map[string]interface{} {
		"ciphertext": ciphertext,
	})
	log.Printf("Decrypted: %+v", decrypted_contents)
	if err != nil {
		log.Fatalf("Error decrypting file: %s", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(decrypted_contents.Data["plaintext"].(string))
	if err != nil {
		log.Fatalf("Error decoding decrypted contents: %s", err)
	}

	return decoded, err
}