package network

const commandSize = 12

type Command [commandSize]byte

var (
	VersionCmd = Command{'v', 'e', 'r', 's', 'i', 'o', 'n', 0, 0, 0, 0, 0}
	VerackCmd  = Command{'v', 'e', 'r', 'a', 'c', 'k', 0, 0, 0, 0, 0, 0}
)

func (c Command) String() string {
	end := commandSize
	for i := 0; i < commandSize; i++ {
		if c[i] == 0 {
			end = i
			break
		}
	}

	return string(c[:end])
}
