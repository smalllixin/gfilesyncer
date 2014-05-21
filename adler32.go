package gfilesyncer

/*
I have to copy & past & mod adler32 algorithm because the hash/adler32 missing rotate method
*/
const (
	// mod is the largest prime that is less than 65536.
	adler32_mod = 65521
	// nmax is the largest n such that
	// 255 * n * (n+1) / 2 + (n+1) * (mod-1) <= 2^32-1.
	// It is mentioned in RFC 1950 (search for "5552").
	adler32_nmax = 5552

	adler32_s1_offset = 1
)

// The size of an Adler-32 checksum in bytes.
const adler32_Size = 4

type AdlerDigest uint32

func (d *AdlerDigest) Reset() { *d = 1 }

// New returns a new hash.Hash32 computing the Adler-32 checksum.
func NewAdler32() *AdlerDigest {
	d := new(AdlerDigest)
	d.Reset()
	return d
}

func (d *AdlerDigest) Size() int { return adler32_Size }

func (d *AdlerDigest) BlockSize() int { return 1 }

// Add p to the running checksum d.
func updateDigest(d AdlerDigest, p []byte) AdlerDigest {
	s1, s2 := uint32(d&0xffff), uint32(d>>16)
	for len(p) > 0 {
		var q []byte
		if len(p) > adler32_nmax {
			p, q = p[:adler32_nmax], p[adler32_nmax:]
		}
		for _, x := range p {
			s1 += uint32(x)
			s2 += s1
		}
		s1 %= adler32_mod
		s2 %= adler32_mod
		p = q
	}
	return AdlerDigest(s2<<16 | s1)
}

func (d *AdlerDigest) Rotate(preByte byte, nextByte byte, distance int) {
	old := uint32(*d)
	s1, s2 := uint32(old&0xffff), uint32(old>>16)

	s1 = s1 - uint32(preByte) + uint32(nextByte);

	s2 = s2 - uint32(int(preByte)* distance) + s1 - adler32_s1_offset;
	s1 %= adler32_mod
	s2 %= adler32_mod
	*d = AdlerDigest(s2<<16 | s1)
}

func (d *AdlerDigest) Write(p []byte) (nn int, err error) {
	*d = updateDigest(*d, p)
	return len(p), nil
}

func (d *AdlerDigest) Sum32() uint32 { return uint32(*d) }

func (d *AdlerDigest) Sum(in []byte) []byte {
	s := uint32(*d)
	return append(in, byte(s>>24), byte(s>>16), byte(s>>8), byte(s))
}

// Checksum returns the Adler-32 checksum of data.
func Checksum(data []byte) uint32 { return uint32(updateDigest(1, data)) }

