package network

const commandSize = 12

type Command [commandSize]byte

var (
	VersionCmd = Command{'v', 'e', 'r', 's', 'i', 'o', 'n', 0, 0, 0, 0, 0}
	VerackCmd  = Command{'v', 'e', 'r', 'a', 'c', 'k', 0, 0, 0, 0, 0, 0}
	PingCmd    = Command{'p', 'i', 'n', 'g', 0, 0, 0, 0, 0, 0, 0, 0}
	PongCmd    = Command{'p', 'o', 'n', 'g', 0, 0, 0, 0, 0, 0, 0, 0}
	GetaddrCmd = Command{'g', 'e', 't', 'a', 'd', 'd', 'r', 0, 0, 0, 0, 0}
	AddrCmd    = Command{'a', 'd', 'd', 'r', 0, 0, 0, 0, 0, 0, 0, 0}
	InvCmd     = Command{'i', 'n', 'v', 0, 0, 0, 0, 0, 0, 0, 0, 0}
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
