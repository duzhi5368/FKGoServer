/***********************************************************
 * D-H秘钥交换
 *
 * Diffie–Hellman key exchange
 *
 * 1. Alice and Bob agree to use a prime number p = 23 and base g = 5.
 *
 * 2. Alice chooses a secret integer a = 6, then sends Bob A = g^a mod p
 * 		A = 5^6 mod 23
 * 		A = 15,625 mod 23
 * 		A = 8
 *
 * 3. Bob chooses a secret integer b = 15, then sends Alice B = g^b mod p
 * 		B = 5^15 mod 23
 * 		B = 30,517,578,125 mod 23
 * 		B = 19
 *
 * 4. Alice computes s = B^a mod p
 * 		s = 19^6 mod 23
 * 		s = 47,045,881 mod 23
 * 		s = 2
 *
 * 5. Bob computes s = A^b mod p
 *	 	s = 8^15 mod 23
 * 		s = 35,184,372,088,832 mod 23
 * 		s = 2
 *
 * 6. Alice and Bob now share a secret (the number 2) because 6 × 15 is the same as 15 × 6
 */
//---------------------------------------------
package dh

//---------------------------------------------
import (
	MATH "math"
	BIG "math/big"
	RAND "math/rand"
	TIME "time"
)

//---------------------------------------------
var (
	rng         = RAND.New(RAND.NewSource(TIME.Now().UnixNano()))
	DH1BASE     = BIG.NewInt(3)
	DH1PRIME, _ = BIG.NewInt(0).SetString("0x7FFFFFC3", 0)
	MAXINT64    = BIG.NewInt(MATH.MaxInt64)
)

//---------------------------------------------
// Diffie-Hellman秘钥交换
func DHExchange() (*BIG.Int, *BIG.Int) {
	SECRET := BIG.NewInt(0).Rand(rng, MAXINT64)
	MODPOWER := BIG.NewInt(0).Exp(DH1BASE, SECRET, DH1PRIME)
	return SECRET, MODPOWER
}

//---------------------------------------------
func DHKey(SECRET, MODPOWER *BIG.Int) *BIG.Int {
	KEY := BIG.NewInt(0).Exp(MODPOWER, SECRET, DH1PRIME)
	return KEY
}

//---------------------------------------------
