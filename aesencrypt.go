package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"bytes"
)

/**
* AES加密
* @plainText 明文
* @key 密钥
* @返回base64加密文本
 */
func AesEncrypt(plainText, key string) (string, error) {
	src := []byte(plainText)
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	bs := block.BlockSize()
	// src = ZeroPadding(src, bs)
	src = PKCS5Padding(src, bs)
	if len(src)%bs != 0 {
		return "", errors.New("need a multiple of the blocksize")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return base64.StdEncoding.EncodeToString(out), nil
}

/**
* AES解密
* @ciphertext 解密数据，base64格式加密文本
* @key 密钥
* 返回解密文本
 */
func AesDecrypt(cipherText, key string) (string, error) {
	src, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	// out = ZeroUnPadding(out)
	out = PKCS5UnPadding(out)
	return string(out), nil
}

/**
* AES加密 ECB模式
* @plainText 明文
* @key 密钥
* @返回hex加密文本
 */
func AesEncryptECB(plainText, key string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", errors.New("key error, please input hex type aes key")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	if plainText == "" {
		return "", errors.New("plainText content empty")
	}

	ecb := NewECBEncrypter(block)
	content := []byte(plainText)
	content = PKCS5Padding(content, block.BlockSize())
	crypted := make([]byte, len(content))
	ecb.CryptBlocks(crypted, content)
	return hex.EncodeToString(crypted), nil
}

/**
* AES解密 ECB模式
* @cipherText 解密数据，hex格式加密文本
* @key 密钥
* @返回解密文本
 */
func AesDecryptECB(cipherText string, key string) (string, error) {
	data, err := hex.DecodeString(cipherText)
	if err != nil {
		return "", errors.New("cipherText error, please input hex type aes cipherText")
	}
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", errors.New("key error, please input hex type aes key")
	}
	cipher, _ := aes.NewCipher([]byte(keyBytes))

	decrypted := make([]byte, len(data))
	size := 16

	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		cipher.Decrypt(decrypted[bs:be], data[bs:be])
	}

	// remove the padding. The last character in the byte array is the number of padding chars
	paddingSize := int(decrypted[len(decrypted)-1])
	return string(decrypted[0 : len(decrypted)-paddingSize]), nil
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

// NewECBEncrypter returns a BlockMode which encrypts in electronic code book
// mode, using the given Block.
func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

type ecbDecrypter ecb

// NewECBDecrypter returns a BlockMode which decrypts in electronic code book
// mode, using the given Block.
func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Decrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
