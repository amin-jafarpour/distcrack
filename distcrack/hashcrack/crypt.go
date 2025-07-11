package hashcrack

/*
#cgo LDFLAGS: -lcrypt
#include <unistd.h>
#include <crypt.h>
#include <stdlib.h>

struct crypt_data* alloc_crypt_data() {
    struct crypt_data* data = (struct crypt_data*) malloc(sizeof(struct crypt_data));
    if (data != NULL) {
        data->initialized = 0;
    }
    return data;
}

void free_crypt_data(struct crypt_data* data) {
    free(data);
}
*/
import "C"
import (
    "fmt"
    "strings"
    "unsafe"
)

// crypt wraps C.crypt_r() and returns the hashed output as a Go string.
func crypt(password, salt string) (string, error) {
    cPassword := C.CString(password)
    cSalt := C.CString(salt)
    defer C.free(unsafe.Pointer(cPassword))
    defer C.free(unsafe.Pointer(cSalt))

    data := C.alloc_crypt_data()
    if data == nil {
        return "", fmt.Errorf("failed to allocate crypt_data")
    }
    defer C.free_crypt_data(data)

    hashed := C.crypt_r(cPassword, cSalt, data)

    if hashed == nil {
        return "", fmt.Errorf("crypt_r returned NULL - invalid input or memory issue")
    }
    return C.GoString(hashed), nil
}

// SplitHash splits a hashed string into its preamble and the hash itself.
func SplitHash(text string) []string {
    tokens := strings.Split(text, "$")
    if len(tokens) > 1 {
        joined := strings.Join(tokens[:len(tokens)-1], "$")
        last := tokens[len(tokens)-1]

        if strings.HasPrefix(text, "$2a$") || strings.HasPrefix(text, "$2b$"){
            return []string{joined + "$" + last[:22], last[22:]} 
        }
        return []string{joined + "$", last}
    }
    fmt.Println("Not enough tokens")
    return nil
}

// isValidSalt checks if the salt format is valid for crypt_r()
func IsValidSalt(salt string) bool {
    return strings.HasPrefix(salt, "$1$") || strings.HasPrefix(salt, "$5$") ||
        strings.HasPrefix(salt, "$6$") || strings.HasPrefix(salt, "$2a$") ||
        strings.HasPrefix(salt, "$2b$") || strings.HasPrefix(salt, "$2y$") ||
        strings.HasPrefix(salt, "$y$")
}

// GenHash generates a password hash using crypt_r()
func GenHash(text string, preamble string) (string, bool) {
    if !IsValidSalt(preamble) {
        fmt.Println("Invalid salt format")
        return "", false
    }
    hash, err := crypt(text, preamble)
    if err != nil {
        fmt.Println("Error hashing:", err)
        return "", false
    }
    return hash, true
}



