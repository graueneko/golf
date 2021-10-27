package golf

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

func TestEmpty(t *testing.T) {
	Reset()
	args := []string{
		"a", "b", "c",
	}
	err := Parse(args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestString(t *testing.T) {
	Reset()
	testConf := "config.yml"
	testPid := "test.pid"
	testKey := "random=str"
	testToken := "1010=10"

	args := []string{
		"-p", testPid,
		"--key", testKey,
	}

	conf := String("c", "conf", "conf_file", "Config File", testConf)
	pid := MustString("p", "", "pid_file", "Pid File")
	key := MustString("", "key", "key", "Key ID")
	token := String("", "token", "token", "Token String", testToken)

	err := Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	if *conf != testConf {
		t.Fatalf("conf: expect %s, got %s", testConf, *conf)
	}
	if testPid != *pid {
		t.Fatalf("pid: expect %s, got %s", testPid, *pid)
	}
	if testKey != *key {
		t.Fatalf("key: expect %s, got %s", testKey, *key)
	}
	if testToken != *token {
		t.Fatalf("token: expect %s, got %s", testToken, *token)
	}

}

func TestUsage(t *testing.T) {
	Reset()
	_ = String("c", "conf", "conf_file", "Config File", "conf.yaml")
	_ = MustString("p", "", "pid_file", "Pid File")
	_ = MustString("", "key", "", "Key ID")
	_ = String("", "token", "", "Token String", "101010")
	_ = BareString("command", "Command To Execute")
	_ = BareString("repeats", "Command Repeat Times")
	_ = BareArray("remains", "Remaining Args")
	expect := `Usage:
  ./test_exec [-c/--conf conf_file] -p pid_file --key string [--token string] command repeats [remains]

Arguments:
  -c/--conf conf_file : Config File (default: "conf.yaml")
  -p pid_file         : Pid File (required)
  --key string        : Key ID (required)
  --token string      : Token String (default: "101010")
  command             : Command To Execute (required)
  repeats             : Command Repeat Times (required)
  remains             : Remaining Args (default: "[]")
`
	str := Usage("./test_exec")
	if str != expect {
		t.Fatalf("Usage info not match")
	}
}

func TestMust(t *testing.T) {
	Reset()
	expect := "missing argument: -s/--server_id server_id"
	MustInt("s", "server_id", "server_id", "Server ID")
	err := Parse([]string{})
	if err == nil {
		t.Fatalf("Expect error, got nil")
	} else if err.Error() != expect {
		t.Fatalf("Expect <%s>, got <%s>", expect, err.Error())
	}
}

func arrayEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestArray(t *testing.T) {
	Reset()
	_ = String("c", "conf", "conf_file", "Config file name", "conf.yml")
	expect := []string{
		"v1", "v2", "v3", "v4",
	}
	vols := Array("v", "volume", "volume", "Volumes")
	args := []string{
		"-v", "v1", "--conf=test_config.yml", "-v=v2", "--volume=v3", "--volume", "v4",
	}

	if err := Parse(args); err != nil {
		t.Fatal(err)
	} else if !arrayEqual(*vols, expect) {
		t.Fatalf("Expect %v, got %v", expect, *vols)
	}
}

func TestInt(t *testing.T) {
	Reset()
	expectRepeat, expectTimes, expectMulti := 1024, 2048, 4096
	repeat := Int("r", "repeat", "repeat_times", "Repeat times", expectRepeat)
	times := MustInt("t", "times", "times", "Times")
	multi := Int("m", "multi", "multiply", "Multiply num", expectMulti)
	args := []string{
		fmt.Sprintf("-t=%d", expectTimes),
		"--multi", strconv.Itoa(expectMulti),
	}

	if err := Parse(args); err != nil {
		t.Fatal(err)
	} else if expectRepeat != *repeat {
		t.Fatalf("Repeat expect %d, got %d", expectRepeat, *repeat)
	} else if expectTimes != *times {
		t.Fatalf("Times expect %d, got %d", expectTimes, *times)
	} else if expectMulti != *multi {
		t.Fatalf("Multi expect %d, got %d", expectMulti, *multi)
	}
}

func TestBool(t *testing.T) {
	Reset()
	expectB1, expectB2, expectB3, expectB4 := true, false, false, true
	b1, b2, b3, b4 := Bool("1", "b1", "", "", expectB1), Bool("2", "b2", "", "", expectB2),
		MustBool("3", "b3", "", ""), MustBool("4", "b4", "", "")
	args := []string{
		"-2", "False", "-3=0", "--b4",
	}
	if err := Parse(args); err != nil {
		t.Fatal(err)
	} else if *b1 != expectB1 || *b2 != expectB2 || *b3 != expectB3 || *b4 != expectB4 {
		t.Fatalf("Expect %v, %v, %v, %v, got %v, %v, %v, %v", expectB1, expectB2, expectB3, expectB4, *b1, *b2, *b3, *b4)
	}
}

func TestParseOSArgs(t *testing.T) {
	Reset()
	os.Args = []string{
		"./test_exec", "-h", "-d",
	}
	if help, err := ParseOSArgs(); err != nil {
		t.Fatal(err)
	} else if !help {
		t.Fatalf("Except Help == true")
	}
	Reset()
	os.Args = []string{
		"./test_exec", "-d",
	}
	d := Bool("d", "daemon", "", "", false)
	if _, err := ParseOSArgs(); err != nil {
		t.Fatal(err)
	} else if *d != true {
		t.Fatalf("Expect d == true, got false")
	}
}
