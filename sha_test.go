package utils

import "testing"

func TestSha1(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "ABC123&abc#"}, want: "2c717085192723aba67a171ac7a67270bf466bc0"},
		{name: "002", args: args{str: "123456"}, want: "7c4a8d09ca3762af61e59520943dc26494f8941b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sha1(tt.args.str); got != tt.want {
				t.Errorf("Sha1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSha256(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "ABC123&abc#"}, want: "a51050f881441cb80edebdf435a7ad1aedb72c05f1fc7becc4bcb0062229b4a7"},
		{name: "002", args: args{str: "123456"}, want: "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sha256(tt.args.str); got != tt.want {
				t.Errorf("Sha256() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSha512(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "001", args: args{str: "ABC123&abc#"}, want: "4f00e3e2532ab5b32f749cacb5523a651d464bc3182baee8f68c3a92a4d6cbdb91a030b1788bf07b502adf6a46fb818cc7ea814cfc988833790e4fecd26fc7e4"},
		{name: "002", args: args{str: "123456"}, want: "ba3253876aed6bc22d4a6ff53d8406c6ad864195ed144ab5c87621b6c233b548baeae6956df346ec8c17f5ea10f35ee3cbc514797ed7ddd3145464e2a0bab413"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sha512(tt.args.str); got != tt.want {
				t.Errorf("Sha512() = %v, want %v", got, tt.want)
			}
		})
	}
}
