//go:build !darwin && !linux

package main

func platformSample(s *Sampler) reading {
	_ = s
	return reading{}
}
