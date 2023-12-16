package main

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/status"
)

// ЗАДАНИЕ:
// * сделать из плохого кода хороший;
// * важно сохранить логику появления ошибочных тасков;
// * сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через merge-request.

// приложение эмулирует получение и обработку тасков, пытается и получать и обрабатывать в многопоточном режиме
// В конце должно выводить успешные таски и ошибки выполнены остальных тасков

// A Ttype represents a meaninglessness of our life
type Ttype struct {
	id         int
	cT         string // время создания
	fT         string // время выполнения
	taskRESULT []byte
}

var ISDEBUG = false

func dprint(d ...any) {
	if ISDEBUG {
		fmt.Println(d...)
	}
}

var n = 10

var TtypePool = sync.Pool{
	New: func() any { return &Ttype{} },
}

func getTtype() *Ttype {
	ifc := TtypePool.Get()
	if ifc != nil {
		return ifc.(*Ttype)
	}
	return TtypePool.New().(*Ttype)
}
func putTtype(b any) {
	TtypePool.Put(b)
}

func main() {
	taskCreturer := func(a chan *Ttype) {
		go func() {
			for {
				ft := time.Now().Format(time.RFC3339)
				if ISDEBUG {
					if time.Now().Second()%2 > 0 { // вот такое условие появления ошибочных тасков
						ft = "Some error occured"
					}
				} else {
					if time.Now().Nanosecond()%2 > 0 { // вот такое условие появления ошибочных тасков
						ft = "Some error occured"
					}
				}
				t := getTtype()
				t.cT = ft
				t.id = int(time.Now().Unix())
				a <- t // передаем таск на выполнение
			}
		}()
	}
	var dCounter any
	if ISDEBUG {
		dCounter = int(0)
	}
	task_worker := func(a *Ttype) *Ttype {
		tt, _ := time.Parse(time.RFC3339, a.cT)
		if tt.After(time.Now().Add(-20 * time.Second)) {
			a.taskRESULT = []byte("task has been successed")
			dprint("ok")
		} else {
			a.taskRESULT = []byte("something went wrong")
			dprint("bad")
			if ISDEBUG {
				dCounter = dCounter.(int)+1
			}
		}
		a.fT = time.Now().Format(time.RFC3339Nano)

		time.Sleep(time.Millisecond * 150)

		return a
	}

	superChan := make(chan *Ttype, n)
	doneTasks := make(chan *Ttype)
	undoneTasks := make(chan error)

	tasksorter := func(t *Ttype) {
		if string(t.taskRESULT[14:]) == "successed" {
			doneTasks <- t
		} else {
			putTtype(t)
			undoneTasks <- status.Errorf(1234, "Task id %d time %s, error %s", t.id, t.cT, t.taskRESULT)
		}
	}

	var mu sync.Mutex
	go func() {
		// получение тасков
	one:
		for {
			select {
			case t, ok := <-superChan:
				if !ok {
					break one
				}
				mu.Lock()
				go tasksorter(task_worker(t))
				mu.Unlock()
			}
		}
		close(superChan)
	}()

	go taskCreturer(superChan)

	var result sync.Map /// := make(map[int]*Ttype,10)
	err := make([]error, 0, n)
	go func() {
	one1:
		for {
			select {
			case r, ok := <-doneTasks:
				if !ok {
					break one1
				}
				result.Store(r.id, r)
			}
		}
		close(doneTasks)
	}()
	go func() {
	one1:
		for {
			select {
			case r, ok := <-undoneTasks:
				if !ok {
					break one1
				}
				err = append(err, r)
			}
		}
		close(undoneTasks)
	}()

	time.Sleep(time.Second * 3)

	dprint("Errors:", dCounter)
	for r := range err {
		println(r)
	}

	dprint("Done tasks:")
	result.Range(func(key, value any) bool {
		dprint(value.(*Ttype).id, string(value.(*Ttype).taskRESULT))
		putTtype(value)
		return true
	})
	// for r := range result {
	// 	println(r)
	// }
}
