package pid

// Controller implements a PID (Proportional-Integral-Derivative) controller.
type Controller struct {
	Kp, Ki, Kd float64
	target     float64
	integral   float64
	lastError  float64
	initialized bool
}

// NewController creates a PID controller with given gains and target.
func NewController(kp, ki, kd, target float64) *Controller {
	return &Controller{Kp: kp, Ki: ki, Kd: kd, target: target}
}

// Adjust computes the PID adjustment given the actual value.
// Returns a multiplier adjustment (e.g. -0.1 means reduce bid by 10%).
func (c *Controller) Adjust(actual float64) float64 {
	if !c.initialized {
		c.initialized = true
		c.lastError = c.target - actual
		return 0
	}
	err := c.target - actual
	c.integral += err
	derivative := err - c.lastError
	c.lastError = err
	return c.Kp*err + c.Ki*c.integral + c.Kd*derivative
}

// Reset clears integral and last error for a fresh start.
func (c *Controller) Reset() {
	c.integral = 0
	c.lastError = 0
	c.initialized = false
}
