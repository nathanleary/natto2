/*
Package natto is an example/offshoot of otto that implements an event loop (supporting setTimeout/setInterval).

http://godoc.org/github.com/robertkrimen/natto

otto: http://github.com/robertkrimen/otto (http://godoc.org/github.com/robertkrimen/otto)

*/
package natto

import (
	"time"

	otto "./otto"
)

type _timer struct {
	timer    *time.Timer
	duration time.Duration
	interval bool
	call     otto.FunctionCall
}

// Run will execute the given JavaScript, continuing to run until all timers have finished executing (if any).
// The VM has the following functions available:
//
//      <timer> = setTimeout(<function>, <delay>, [<arguments...>])
//      <timer> = setInterval(<function>, <delay>, [<arguments...>])
//      clearTimeout(<timer>)
//      clearInterval(<timer>)
//

func Run(src string, vm *otto.Otto) error {

	//vm := otto.New()
	registry := map[*_timer]*_timer{}
	ready := make(chan *_timer)

	newTimer := func(call otto.FunctionCall, interval bool) (*_timer, otto.Value) {
		delay, _ := call.Argument(1).ToFloat()
		if 0 >= delay {
			delay = 1
		}

		timer := &_timer{
			duration: time.Duration(delay) * time.Millisecond,
			call:     call,
			interval: interval,
		}
		registry[timer] = timer

		timer.timer = time.AfterFunc(timer.duration, func() {
			ready <- timer
		})

		value, err := call.Otto.ToValue(timer)
		if err != nil {
			panic(err)
		}

		return timer, value
	}

	setTimeout := func(call otto.FunctionCall) otto.Value {
		_, value := newTimer(call, false)
		return value
	}
	vm.Set("setTimeout", setTimeout)

	setInterval := func(call otto.FunctionCall) otto.Value {
		_, value := newTimer(call, true)
		return value
	}
	vm.Set("setInterval", setInterval)

	clearTimeout := func(call otto.FunctionCall) otto.Value {
		timer, _ := call.Argument(0).Export()
		if timer, ok := timer.(*_timer); ok {
			timer.timer.Stop()
			delete(registry, timer)
		}
		return otto.UndefinedValue()
	}
	vm.Set("clearTimeout", clearTimeout)
	vm.Set("clearInterval", clearTimeout)
	//

	//wg := sync.WaitGroup{}
	//wg.Add(1)
	var err error
	//go func() {
	//defer wg.Done()
	//_, err = vm.Run(src)

	//}()

	//wg.Wait()

	if err != nil {
		return err
	}
	//	wg := sync.WaitGroup{}
	//	wg.Add(1)
	//var err2 error
	//	go func() {
	//	defer wg.Done()
	for {
		select {
		case timer := <-ready:
			var arguments []interface{}
			if len(timer.call.ArgumentList) > 2 {
				tmp := timer.call.ArgumentList[2:]
				arguments = make([]interface{}, 2+len(tmp))
				for i, value := range tmp {
					arguments[i+2] = value
				}
			} else {
				arguments = make([]interface{}, 1)
			}
			arguments[0] = timer.call.ArgumentList[0]
			_, err := vm.Call(`Function.call.call`, nil, arguments...)
			if err != nil {
				for _, timer := range registry {

					timer.timer.Stop()
					delete(registry, timer)
					//err2 = err
					return err
				}
			}
			if timer.interval {
				timer.timer.Reset(timer.duration)
			} else {
				delete(registry, timer)
			}
		default:
			// Escape valve!
			// If this isn't here, we deadlock...
		}
		if len(registry) == 0 {
			break
		}
	}
	_, err = vm.Run("_jump(function() {\n" + src + "\n});")
	//}()
	//wg.Wait()
	return err
}
