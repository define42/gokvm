package main

import (
	"log"
	"os"

	"github.com/bobuhiro11/gokvm/flag"
	"github.com/bobuhiro11/gokvm/probe"
	"github.com/bobuhiro11/gokvm/vmm"
)

func main() {
	bootArgs, probeArgs, err := flag.ParseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if bootArgs != nil {
		c := &vmm.Config{
			Dev:            bootArgs.Dev,
			Kernel:         bootArgs.Kernel,
			Initrd:         bootArgs.Initrd,
			ISO:            bootArgs.ISO,
			Params:         bootArgs.Params,
			ParamsSet:      bootArgs.ParamsSet,
			TapIfName:      bootArgs.TapIfName,
			Network:        bootArgs.Network,
			Disk:           bootArgs.Disk,
			GPU:            bootArgs.GPU,
			VNC:            bootArgs.VNC,
			RDP:            bootArgs.RDP,
			RDPCert:        bootArgs.RDPCert,
			RDPKey:         bootArgs.RDPKey,
			RDPH264:        bootArgs.RDPH264,
			RDPH264Threads: bootArgs.RDPH264Threads,
			RDPStats:       bootArgs.RDPStats,
			Audio:          bootArgs.Audio,
			NCPUs:          bootArgs.NCPUs,
			MemSize:        bootArgs.MemSize,
			TraceCount:     bootArgs.TraceCount,
		}

		vmm := vmm.New(*c)

		if err := vmm.Init(); err != nil {
			log.Fatal(err)
		}

		if err := vmm.Setup(); err != nil {
			log.Fatal(err)
		}

		if err := vmm.Boot(); err != nil {
			log.Fatal(err)
		}
	}

	if probeArgs != nil {
		if err := probe.KVMCapabilities(); err != nil {
			log.Fatal(err)
		}

		if err := probe.CPUID(); err != nil {
			log.Fatal(err)
		}
	}
}
