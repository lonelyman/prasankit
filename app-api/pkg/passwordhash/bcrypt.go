package passwordhash

import "golang.org/x/crypto/bcrypt"

const DefaultBcryptCost = 12

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) BcryptHasher {
	if cost == 0 {
		cost = DefaultBcryptCost
	}
	return BcryptHasher{
		cost: cost,
	}
}

func (h BcryptHasher) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (h BcryptHasher) Compare(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func IsBcryptHash(hash string) bool {
	_, err := bcrypt.Cost([]byte(hash))
	return err == nil
}
