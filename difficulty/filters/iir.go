// IIR filter generated on: 2015-05-11 15:39:02

package filters

// input parameters:
//   filter type:                elliptic
//   passband frequency finish:   14.0000000000000000 Hz
//   stopband frequency start:    16.0000000000000000 Hz
//   Sampling frequency:         100.0000000000000000 Hz
//   passband ripple:              0.1000000000000000 dB
//   stopband attuation:          32.0000000000000000 dB
//   comparison limit:             0.0000000000000100

// computed values:
//   filter order (n):             6
//   pass band cutoff freq. Wp:    0.2800000000000000
//   stop band edge freq. Ws:      0.3200000000000000
//   cutoff frequency     Wc:      0.2800000000000000

//   filter coefficients a:
// 	  1.0000000000000000     	 -3.3736258843777711     	  5.7929873873369164     	 -5.8754785372511815
// 	  3.7251352397391848     	 -1.3689325851656009     	  0.2335420568989748

//   filter coefficients b:
// 	  0.0474787642386640     	 -0.0587471418358881     	  0.1197625812644266     	 -0.0848903535257803
// 	  0.1197625812644266     	 -0.0587471418358881     	  0.0474787642386640

//   filter expression:
//     a(1)y(n) = b(1)x(n)   + b(2)x(n-1) + ... + b(nb+1)x(n-nb)
//              - a(2)y(n-1) - a(3)y(n-2) - ... - a(na+1)y(n-na)

// IIR - infinite impulse response filter
type IIR struct {
	x [7]float64
	y [7]float64
}

// Filter - IIR with loops unrolled
func (f *IIR) Filter(x float64) float64 {

	f.x[0] = f.x[1]
	f.x[1] = f.x[2]
	f.x[2] = f.x[3]
	f.x[3] = f.x[4]
	f.x[4] = f.x[5]
	f.x[5] = f.x[6]
	f.x[6] = x // new sample

	f.y[0] = f.y[1]
	f.y[1] = f.y[2]
	f.y[2] = f.y[3]
	f.y[3] = f.y[4]
	f.y[4] = f.y[5]
	f.y[5] = f.y[6]

	f.y[6] =
		+0.0474787642386640*(f.x[6]+f.x[0]) +
			-0.0587471418358881*(f.x[5]+f.x[1]) +
			+0.1197625812644266*(f.x[4]+f.x[2]) +
			-0.0848903535257803*f.x[3] +
			+3.3736258843777711*f.y[5] +
			-5.7929873873369164*f.y[4] +
			+5.8754785372511815*f.y[3] +
			-3.7251352397391848*f.y[2] +
			+1.3689325851656009*f.y[1] +
			-0.2335420568989748*f.y[0] // - a(7)y(0)
	return f.y[6]
}
