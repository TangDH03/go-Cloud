package mr

import (
	"cloud/dao"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const waitTime = 100 * time.Millisecond
const (
	Idle = iota + 1<<4
	Busy
	IsWriting
	Success
	WriteFail
)

var once sync.Once
var uploadTask sync.Map
var taskWaitGroup sync.Map
var m *Master

type Master struct {
	workers []*Worker
	mu      sync.Mutex
}

func (master *Master) WorkersNum() int {
	return len(master.workers)
}

func (master *Master) Send(result chan bool, job interface{}) {
	switch job.(type) {
	case UploadJob:

		var waitg sync.WaitGroup
		master.mu.Lock()
		wg, ok := taskWaitGroup.Load(job.(UploadJob).Md5sum)
		if !ok {
			taskWaitGroup.Store(job.(UploadJob), &waitg)
			waitg.Add(1)
			master.mu.Unlock()
			success := master.sendUpload(job.(UploadJob), &waitg)
			waitg.Wait()

			value, exist := uploadTask.Load(job.(UploadJob).Md5sum)
			fmt.Fprintln(gin.DefaultWriter,
				fmt.Sprintf("determine recover or not value: %v exist: %v", value, exist))

			if !success && exist && value == Success { //recover task

				result <- dao.FastUpload(job.(UploadJob).Md5sum,
					job.(UploadJob).File.Filename, job.(UploadJob).UsrName)
				fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
				//result <- true
				return
			}

			fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
			result <- (exist && value == Success)
		} else {
			wg.(*sync.WaitGroup).Add(1)
			master.mu.Unlock()
			success := master.sendUpload(job.(UploadJob), wg.(*sync.WaitGroup))
			wg.(*sync.WaitGroup).Wait()
			value, exist := uploadTask.Load(job.(UploadJob).Md5sum)
			if !success && exist && value == Success { //recover task
				result <- dao.FastUpload(job.(UploadJob).Md5sum,
					job.(UploadJob).File.Filename, job.(UploadJob).UsrName)

				fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
				return
			}

			fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
			result <- (exist && value == Success)
		}
	case UploadBigJob:

		var waitg sync.WaitGroup
		master.mu.Lock()
		wg, ok := taskWaitGroup.Load(job.(UploadBigJob).Md5sum)
		if !ok {
			taskWaitGroup.Store(job.(UploadBigJob), &waitg)
			waitg.Add(1)
			master.mu.Unlock()
			success := master.sendBigUpload(job.(UploadBigJob), &waitg)
			waitg.Wait()

			value, exist := uploadTask.Load(job.(UploadBigJob).Md5sum)
			fmt.Fprintln(gin.DefaultWriter,
				fmt.Sprintf("determine recover or not value: %v exist: %v", value, exist))

			if !success && exist && value == Success { //recover task

				result <- dao.FastTmpUpload(job.(UploadBigJob).Md5sum,
					job.(UploadBigJob).File.Filename,
					job.(UploadBigJob).UsrName, job.(UploadBigJob).Fragment)
				fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
				//result <- true
				return
			}

			fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
			result <- (exist && value == Success)
		} else {
			wg.(*sync.WaitGroup).Add(1)
			master.mu.Unlock()
			success := master.sendBigUpload(job.(UploadBigJob), wg.(*sync.WaitGroup))
			wg.(*sync.WaitGroup).Wait()
			value, exist := uploadTask.Load(job.(UploadBigJob).Md5sum)
			if !success && exist && value == Success { //recover task
				result <- dao.FastTmpUpload(job.(UploadBigJob).Md5sum,
					job.(UploadBigJob).File.Filename,
					job.(UploadBigJob).UsrName, job.(UploadBigJob).Fragment)
				fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
				return
			}
			fmt.Fprintln(gin.DefaultWriter, "master send reply to caller")
			result <- (exist && value == Success)
		}
	default:
		result <- true

	}
}

//initial Call First
func initial() {
	m = &Master{}
	m.workers = make([]*Worker, 100)
	for i := 0; i < len(m.workers); i++ {
		jobs := make(chan interface{}, 2)
		reply := make(chan interface{}, 2)
		m.workers[i] = &Worker{State: Idle,
			Jobs: jobs, Reply: reply}
		go m.workers[i].Run()
	}
}

func GetMaster() *Master {
	once.Do(initial)
	return m
}

func (master *Master) pickUpIdle() *Worker {

GetIdle:
	master.mu.Lock()
	for i := 0; i < len(master.workers); i++ {
		if master.workers[i].State == Idle {
			master.setBusy(master.workers[i])
			fmt.Fprintln(gin.DefaultWriter,
				fmt.Sprintf("master receive a job ,give worker:[%v]", i))

			master.mu.Unlock()
			return master.workers[i]
		}
	}
	master.mu.Unlock()
	time.Sleep(waitTime)
	goto GetIdle

}

//not thread safe
func (master *Master) setIdle(worker *Worker) {
	worker.State = Idle
}

//not thread safe
func (master *Master) setBusy(worker *Worker) {
	worker.State = Busy
}

func (master *Master) sendUpload(job UploadJob, wg *sync.WaitGroup) bool {
	//just a tricky
	var code int
	defer wg.Done()
WaitComplete:
	for {
		master.mu.Lock()
		value, ok := uploadTask.Load(job.Md5sum)
		master.mu.Unlock()
		if ok {
			code = value.(int)
		}
		if ok && code == IsWriting {
			time.Sleep(waitTime)
		} else {
			break
		}
	}
	if code == Success {
		dao.FastUpload(job.Md5sum,
			job.File.Filename, job.UsrName)
		return true
	} else if code == WriteFail {
		master.mu.Lock()
		value, _ := uploadTask.Load(job.Md5sum)
		if value == IsWriting {
			master.mu.Unlock()
			goto WaitComplete
		}
		uploadTask.Store(job.Md5sum, IsWriting)
		master.mu.Unlock()
	}
	master.mu.Lock()
	value, ok := uploadTask.Load(job.Md5sum)
	if ok && value.(int) == IsWriting {
		master.mu.Unlock()
		goto WaitComplete
	}
	if ok && value.(int) == Success {
		master.mu.Unlock()
		dao.FastUpload(job.Md5sum,
			job.File.Filename, job.UsrName)
		return true
	}
	uploadTask.Store(job.Md5sum, IsWriting)
	master.mu.Unlock()
	worker := master.pickUpIdle()
	worker.Jobs <- job
	reply := <-worker.Reply
	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("worker complete reply is %v", reply))
	if reply.(bool) {
		master.mu.Lock()
		uploadTask.Store(job.Md5sum, Success)
		master.setIdle(worker)
		master.mu.Unlock()
		return true
	} else {
		master.mu.Lock()
		uploadTask.Store(job.Md5sum, WriteFail)
		master.setIdle(worker)
		master.mu.Unlock()
		return false
	}

}
func (master *Master) sendBigUpload(job UploadBigJob, wg *sync.WaitGroup) bool {
	//just a tricky
	var code int
	defer wg.Done()
WaitComplete:
	for {
		master.mu.Lock()
		value, ok := uploadTask.Load(job.Md5sum)
		master.mu.Unlock()
		if ok {
			code = value.(int)
		}
		if ok && code == IsWriting {
			time.Sleep(waitTime)
		} else {
			break
		}
	}
	if code == Success {
		dao.FastUpload(job.Md5sum,
			job.File.Filename, job.UsrName)
		return true
	} else if code == WriteFail {
		master.mu.Lock()
		value, _ := uploadTask.Load(job.Md5sum)
		if value == IsWriting {
			master.mu.Unlock()
			goto WaitComplete
		}
		uploadTask.Store(job.Md5sum, IsWriting)
		master.mu.Unlock()
	}
	master.mu.Lock()
	value, ok := uploadTask.Load(job.Md5sum)
	if ok && value.(int) == IsWriting {
		master.mu.Unlock()
		goto WaitComplete
	}
	if ok && value.(int) == Success {
		master.mu.Unlock()
		dao.FastTmpUpload(job.Md5sum,
			job.File.Filename, job.UsrName, job.Fragment)
		return true
	}
	uploadTask.Store(job.Md5sum, IsWriting)
	master.mu.Unlock()
	worker := master.pickUpIdle()
	worker.Jobs <- job
	reply := <-worker.Reply
	fmt.Fprintln(gin.DefaultWriter, fmt.Sprintf("worker complete reply is %v", reply))
	if reply.(bool) {
		master.mu.Lock()
		uploadTask.Store(job.Md5sum, Success)
		master.setIdle(worker)
		master.mu.Unlock()
		return true
	} else {
		master.mu.Lock()
		uploadTask.Store(job.Md5sum, WriteFail)
		master.setIdle(worker)
		master.mu.Unlock()
		return false
	}

}
