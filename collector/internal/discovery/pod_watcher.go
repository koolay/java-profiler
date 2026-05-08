package discovery

type PodMetadata struct {
	Namespace   string
	Workload    string
	Pod         string
	PodUID      string
	Container   string
	Node        string
	Service     string
	Labels      map[string]string
	Annotations map[string]string
}

type PodWatcher struct {
	pods map[string]PodMetadata
}

func NewPodWatcher() *PodWatcher {
	return &PodWatcher{pods: map[string]PodMetadata{}}
}

func (w *PodWatcher) Upsert(pod PodMetadata) {
	w.pods[pod.PodUID+"/"+pod.Container] = pod
}

func (w *PodWatcher) Resolve(podUID, container string) (PodMetadata, bool) {
	pod, ok := w.pods[podUID+"/"+container]
	return pod, ok
}
