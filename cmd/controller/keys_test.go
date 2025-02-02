package main

import (
	"crypto/rsa"
	"crypto/x509"
	"io"
	mathrand "math/rand"
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	certUtil "k8s.io/client-go/util/cert"
)

// This is omg-not safe for real crypto use!
func testRand() io.Reader {
	return mathrand.New(mathrand.NewSource(42))
}

func TestReadKey(t *testing.T) {
	rand := testRand()

	key, err := rsa.GenerateKey(rand, 512)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	cert, err := signKey(rand, key)
	if err != nil {
		t.Fatalf("Failed to self-sign key: %v", err)
	}

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mykey",
			Namespace: "myns",
		},
		Data: map[string][]byte{
			v1.TLSPrivateKeyKey: certUtil.EncodePrivateKeyPEM(key),
			v1.TLSCertKey:       certUtil.EncodeCertPEM(cert),
		},
		Type: v1.SecretTypeTLS,
	}

	key2, cert2, err := readKey(secret)
	if err != nil {
		t.Errorf("readKey() failed with: %v", err)
	}

	if !reflect.DeepEqual(key, key2) {
		t.Errorf("Extracted key != original key")
	}

	if !reflect.DeepEqual(cert, cert2[0]) {
		t.Errorf("Extracted cert != original cert")
	}
}

func TestWriteKey(t *testing.T) {
	rand := testRand()
	key, err := rsa.GenerateKey(rand, 512)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	cert, err := signKey(rand, key)
	if err != nil {
		t.Fatalf("signKey failed: %v", err)
	}

	client := fake.NewSimpleClientset()

	_, err = writeKey(client, key, []*x509.Certificate{cert}, "myns", "label", "mykey")
	if err != nil {
		t.Errorf("writeKey() failed with: %v", err)
	}

	t.Logf("actions: %v", client.Actions())

	if a := findAction(client, "create", "secrets"); a == nil {
		t.Errorf("writeKey didn't create a secret")
	} else if a.GetNamespace() != "myns" {
		t.Errorf("writeKey() created key in wrong namespace!")
	}
}

func TestSignKey(t *testing.T) {
	rand := testRand()

	key, err := rsa.GenerateKey(rand, 512)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	cert, err := signKey(rand, key)
	if err != nil {
		t.Errorf("signKey() returned error: %v", err)
	}

	if !reflect.DeepEqual(cert.PublicKey, &key.PublicKey) {
		t.Errorf("cert pubkey != original pubkey")
	}
}
