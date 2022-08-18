package dotenv

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func EnsureSet(vals ...string) {
	for _, v := range vals {
		if _, found := os.LookupEnv(v); !found {
			log.Fatalf("'%v' not set in .env or environment", v)
		}
	}
}

// NOT SUPPORTED: Overriding Env variables, Multiline strings
func init() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "#") || len(line) == 1 {
			continue
		}

		fields := strings.Split(line, "=")
		if len(fields) != 2 {
			continue
		}

		key := strings.TrimSpace(fields[0])
		val := strings.TrimSpace(fields[1])
		if val[0] == '"' || val[0] == '\'' {
			quote := val[0]
			if val[len(val)-1] != quote {
				continue
			}
		}

		if _, set := os.LookupEnv(key); set {
			continue
		}
		os.Setenv(key, val)
	}

}
