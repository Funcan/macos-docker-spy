package main

import "testing"

func TestSummarise(t *testing.T) {
	tests := []struct {
		name, out, want string
	}{
		{
			name: "real VM output",
			out: `/dev/vda1       32846608 27429352   3723204  89% /var/lib/docker
Mem:        16356724      638168    15646468         508      270780    15718556
0.36 0.08 0.03 3/253 1489
cpu  194 0 118 67514 57 0 104 0 0 0
cpu  194 0 119 68317 57 0 104 0 0 0
`,
			// Golden output from the shell/awk version of the probe.
			want: "Disk 27 GB of 32 GB used (89%)  3.6 GB free | 624 MB used 265 MB buff/cache 15 GB free | load 0.36 0.08 0.03 | cpu 0% usr 0% sys 99% idle",
		},
		{
			name: "busy CPU and small sizes",
			out: `/dev/vda1 900000 500 399500 1% /var/lib/docker
Mem: 2000 1100 800 0 100 900
1.00 2.00 3.00 1/1 1
cpu  100 10 50 800 40 5 5 3 0 0
cpu  300 30 100 1200 60 10 15 7 0 0
`,
			want: "Disk 1 MB of 879 MB used (1%)  391 MB free | 2 MB used 1 MB buff/cache 1 MB free | load 1.00 2.00 3.00 | cpu 31% usr 9% sys 59% idle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := summarise(tt.out)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestSummariseBadOutput(t *testing.T) {
	if _, err := summarise("nsenter: failed\n"); err == nil {
		t.Error("expected an error")
	}
}
