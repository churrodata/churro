// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package extractsource

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	testclient "k8s.io/client-go/kubernetes/fake"
	cruntime "sigs.k8s.io/controller-runtime/pkg/client/config"

	//	"github.com/fsnotify/fsnotify"
	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/db"
	"github.com/churrodata/churro/internal/pipeline"
	"github.com/churrodata/churro/pkg"
	"github.com/churrodata/churro/pkg/config"
	pb "github.com/churrodata/churro/rpc/extractsource"

	"os"

	"github.com/rs/xid"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

// DefaultPort is the extractsource service port
const DefaultPort = ":8087"

type QueueEntry struct {
	filePath string
	dirPath  string
	regex    string
}

// Server implements the extractsource service
type Server struct {
	Pi           v1alpha1.Pipeline
	ServiceCreds config.ServiceCredentials
	DBCreds      config.DBCredentials
	UserDBCreds  config.DBCredentials
	//	Watcher      *fsnotify.Watcher
	QueueOfFiles chan QueueEntry
}

// Ping ...
func (s *Server) Ping(ctx context.Context, size *pb.PingRequest) (hat *pb.PingResponse, err error) {
	return &pb.PingResponse{}, nil
}

// NewExtractSourceServer creates a extractsource server and returns
// a pointer to it.  The server is built based on the configuration
// passed to it.
func NewExtractSourceServer(debug bool, svcCreds config.ServiceCredentials, pipeline v1alpha1.Pipeline, userDBCreds config.DBCredentials, dbCreds config.DBCredentials) *Server {

	s := &Server{
		ServiceCreds: svcCreds,
		UserDBCreds:  userDBCreds,
		DBCreds:      dbCreds,
		Pi:           pipeline,
		QueueOfFiles: make(chan QueueEntry),
	}

	go s.startQueueConsumer()

	//go s.startWatching()
	go s.startINotify()

	log.Info().Msg("extractsource service started on port " + DefaultPort)
	return s
}

func getTable(filePath, scheme string, dirs []v1alpha1.ExtractSourceDefinition) (string, string, error) {
	baseDir := filepath.Dir(filePath)

	for _, dir := range dirs {
		if dir.Scheme == scheme && strings.Contains(baseDir, dir.Path) {
			return dir.Name, dir.Tablename, nil
		}
	}
	return "", "", fmt.Errorf("could not find right scheme %s and dir %s in churro config", scheme, baseDir)
}

func (s *Server) createExtractPod(client kubernetes.Interface, scheme string, filePath string, cfg v1alpha1.Pipeline, tableName, extractSourceName string) error {
	imageName := os.Getenv("CHURRO_EXTRACT_IMAGE")
	ns := os.Getenv("CHURRO_NAMESPACE")
	pipelineName := os.Getenv("CHURRO_PIPELINE")
	ctx := context.TODO()

	// TODO cleanup this selection logic as its redundant

	switch scheme {
	case extractapi.APIScheme:
	case extractapi.XMLScheme:
	case extractapi.CSVScheme:
	case extractapi.JSONScheme:
	case extractapi.JSONPathScheme:
	case extractapi.XLSXScheme:
		log.Debug().Msg("scheme used for extract job " + scheme)
	default:
		return fmt.Errorf("%s scheme is not recognized", scheme)
	}

	/**
	client, err := GetKubeClient("")
	if err != nil {
		return err
	}
	*/

	pod := getPodDefinition(filePath, tableName, scheme, rand.String(4), ns, imageName, pipelineName, extractSourceName)
	log.Debug().Msg("creating pod " + pod.Name)

	_, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// getPodDefinition fills out a Pod definition
func getPodDefinition(filePath, tableName, scheme, suffix, namespace, imageName, pipelineName, extractSourceName string) *v1.Pod {
	entrypoint := []string{
		"/usr/local/bin/churro-extract",
		"-servicecert",
		"/servicecerts",
		"-dbcert",
		"/dbcerts",
		"-debug",
		"true",
	}

	var mode int32
	//mode = 0620
	mode = 256

	extractLogID := xid.New().String()

	pp := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("churro-extract-%s", suffix),
			Namespace: namespace,
			Labels: map[string]string{
				"app":          "churro",
				"service":      "churro-extract",
				"watchdirname": extractSourceName,
				"extractlogid": extractLogID,
			},
		},
		Spec: v1.PodSpec{
			ServiceAccountName: "churro",
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:            "churro-extract",
					Image:           imageName,
					ImagePullPolicy: v1.PullIfNotPresent,
					Command:         entrypoint,
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/dbcerts",
							Name:      "db-certs",
							ReadOnly:  true,
						},
						{
							MountPath: "/servicecerts",
							Name:      "service-certs",
							ReadOnly:  true,
						},
						{
							MountPath: "/churro",
							Name:      "churrodata",
							ReadOnly:  false,
						},
					},
					Env: []v1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name: "CHURRO_NAMESPACE",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
						{
							Name:  "CHURRO_EXTRACTLOG",
							Value: extractLogID,
						},
						{
							Name:  "CHURRO_PIPELINE",
							Value: pipelineName,
						},
						{
							Name:  "CHURRO_FILENAME",
							Value: filePath,
						},
						{
							Name:  "CHURRO_SCHEME",
							Value: scheme,
						},
						{
							Name:  "CHURRO_WATCHDIR_NAME",
							Value: extractSourceName,
						},
						{
							Name:  "CHURRO_TABLENAME",
							Value: tableName,
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "db-certs",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName:  "cockroachdb.client.root",
							DefaultMode: &mode,
						},
					},
				},
				{
					Name: "service-certs",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "churro.client.root",
						},
					},
				},
				{
					Name: "churrodata",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: "churrodata",
						},
					},
				},
			},
		},
	}

	pullSecretName := os.Getenv("CHURRO_PULL_SECRET_NAME")
	if pullSecretName != "" {
		ref := v1.LocalObjectReference{
			Name: pullSecretName,
		}
		pp.Spec.ImagePullSecrets = []v1.LocalObjectReference{ref}
	}

	return pp
}

// GetKubeClient gets a connection to the Kube cluster
func GetKubeClient(kubeconfig string) (client kubernetes.Interface, err error) {

	if os.Getenv("FAKECLIENT") != "" {
		return testclient.NewSimpleClientset(), nil
	}

	config, err := cruntime.GetConfig()
	if err != nil {
		return client, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, err
	}

	return clientset, err
}

func (s *Server) createExtractSources() {

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting kubeclientset err ")
		return
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return
	}

	// set up a file watcher for all non-API extract sources
	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		dir := pipelineToUpdate.Spec.Extractsources[i]

		log.Debug().Msg("extractsource service: watching " + dir.Path)
		if dir.Scheme == extractapi.APIScheme {
			log.Info().Msg("skipping dir setup for api scheme")
			continue
		}

		_, err := os.Stat(dir.Path)
		if os.IsNotExist(err) {
			log.Error().Stack().Err(err).Msg("dir path not exist, will create " + dir.Path)
			err = os.Mkdir(dir.Path, os.ModePerm)
			if err != nil {
				log.Error().Stack().Err(err).Msg("could not create directory " + dir.Path)
			} else {
				log.Info().Msg("created directory " + dir.Path)
			}
			err = os.Mkdir(dir.Path+"/ready", os.ModePerm)
			if err != nil {
				log.Error().Stack().Err(err).Msg("could not create ready directory " + dir.Path)
			} else {
				log.Info().Msg("created directory " + dir.Path)
			}
		}
		_, err = os.Stat(dir.Path)
		if err == nil {
			//err = s.Watcher.Add(dir.Path)
			go s.watchEventsFor(dir.Path, dir.Regex)
			//if err != nil {
			//	log.Error().Stack().Err(err).Msg("error adding watch path " + dir.Path)
			//} else {
			log.Info().Msg("added inotify watched path " + dir.Path)
			//}
		}
	}
}

func (s *Server) createExtractPodForNewFile(dirPath, filePath, regex string) error {
	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("error getting kubeclientset err ")
		return err
	}

	pipelineClient, err := pkg.NewClient(config, s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	pipelineToUpdate, err := pipelineClient.Get(s.Pi.Name)
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
		return err
	}

	for i := 0; i < len(pipelineToUpdate.Spec.Extractsources); i++ {
		c := pipelineToUpdate.Spec.Extractsources[i]
		if c.Scheme == extractapi.APIScheme {
			continue
		}

		// match the extractsource using the dirpath which is unique
		if dirPath == c.Path {

			scheme := c.Scheme
			log.Info().Msg("dir " + dirPath + "scheme " + scheme + " regex " + regex)

			extractSourceName, tableName, err := getTable(filePath, c.Scheme, pipelineToUpdate.Spec.Extractsources)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error getting table")
				return err
			}
			otherClient, err := GetKubeClient("")
			if err != nil {
				log.Error().Stack().Err(err).Msg("error getting otherclient")
				return err
			}

			err = s.createExtractPod(otherClient, c.Scheme, filePath, s.Pi, tableName, extractSourceName)
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in createExtractPod ")
				return err
			}

			// return, we found the matching watchdir based on scheme
			// and filePath matching
			return nil
		}
	}
	return nil
}

// CreateExtractSource ....
func (s *Server) CreateExtractSource(ctx context.Context, req *pb.CreateExtractSourceRequest) (response *pb.CreateExtractSourceResponse, err error) {

	resp := &pb.CreateExtractSourceResponse{}
	log.Info().Msg("createExtractSources called")
	s.createExtractSources()
	return resp, nil
}

// DeleteExtractSource ...
func (s *Server) DeleteExtractSource(ctx context.Context, req *pb.DeleteExtractSourceRequest) (response *pb.DeleteExtractSourceResponse, err error) {

	resp := &pb.DeleteExtractSourceResponse{}

	// TODO update the current in-memory configuration by
	// removing the passed WatchName req.WatchName
	if req.ExtractSourceName == "" {
		log.Error().Stack().Msg("error extract source name is empty")
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return resp, nil
}

// GetExtractSourceServiceConnection ...
func GetExtractSourceServiceConnection(namespace string) (client pb.ExtractSourceClient, err error) {

	// get the service.crt from the pipeline CustomResource

	var p v1alpha1.Pipeline
	p, err = pipeline.GetPipeline(namespace)
	if err != nil {
		return client, err
	}

	serviceCrt1 := p.Spec.ServiceCredentials.ServiceCrt
	url := fmt.Sprintf("churro-extractsource.%s.svc.cluster.local%s", namespace, DefaultPort)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(serviceCrt1))
	creds := credentials.NewClientTLSFromCert(caCertPool, "")

	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(creds))
	if err != nil {
		return client, err
	}

	client = pb.NewExtractSourceClient(conn)

	return client, nil

}

func (s Server) bumpFilesMetric() {
}

func (s *Server) startQueueConsumer() {
	log.Info().Msg("processing the queue...")
	clientset, _, err := pkg.GetKubeClient()
	if err != nil {
		panic(err)
	}

	for f := range s.QueueOfFiles {

		podCount, err := getExtractCount(clientset, s.Pi.Name)
		if err != nil {
			log.Error().Stack().Err(err)
		}

		// sleep a bit if the max jobs is reached, then requeue the
		// original file name to retry
		if podCount >= s.Pi.Spec.MaxJobs {
			log.Error().Msg(fmt.Sprintf("WARNING:  max jobs reached %d  max set to %d requeuing\n", podCount, s.Pi.Spec.MaxJobs))
			time.Sleep(5 * time.Second)
			s.QueueOfFiles <- f
			continue
		}
		log.Info().Msg(fmt.Sprintf("working on file %s regex %s current jobs %d\n", f.filePath, f.regex, podCount))
		err = s.createExtractPodForNewFile(f.dirPath, f.filePath, f.regex)
		if err != nil {
			log.Error().Stack().Err(err)
		}
	}
}

//func getExtractCount(clientset *kubernetes.Clientset, namespace string) (count int, err error) {
func getExtractCount(clientset kubernetes.Interface, namespace string) (count int, err error) {
	labelSelector := fmt.Sprintf("service=churro-extract")
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return count, err
	}
	var running int
	for i := 0; i < len(pods.Items); i++ {
		p := pods.Items[i]
		if p.Status.Phase == v1.PodRunning {
			running++
		}
	}

	log.Info().Msg(fmt.Sprintf("extract pods running %d\n", running))
	return running, nil
}

func (s *Server) testDatabase() {

	databaseMaxTries := 7
	for i := 0; i < databaseMaxTries; i++ {

		churroDB, err := db.NewChurroDB(s.Pi.Spec.DatabaseType)
		if err != nil {
			if i >= databaseMaxTries {
				log.Error().Stack().Err(err).Msg("could not connect to db giving up")
				os.Exit(1)
			}
			log.Error().Stack().Err(err).Msg("error db ")
			time.Sleep(time.Second * 10)
			continue
		}

		err = churroDB.GetConnection(s.DBCreds, s.Pi.Spec.AdminDataSource)
		if err != nil {
			if i >= databaseMaxTries {
				log.Error().Stack().Err(err).Msg("error opening pipeline db")
				os.Exit(1)
			}
			time.Sleep(time.Second * 10)
			continue
		}

		_, err = churroDB.GetAllPipelineMetrics()
		if err != nil {
			if i >= databaseMaxTries {
				log.Error().Stack().Err(err).Msg("error getting watch directories giving up")
				os.Exit(1)
			}
			time.Sleep(time.Second * 10)
			continue
		}
		break
	}
	log.Info().Msg("connected to database")

}

func (s *Server) startINotify() {

	s.createExtractSources()

}

func (s *Server) watchEventsFor(dir, regex string) {

	fd, err := unix.InotifyInit1(0)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inotify on " + dir)
		return
	}
	defer unix.Close(fd)

	_, err = unix.InotifyAddWatch(
		fd,
		dir,
		unix.IN_CREATE|
			unix.IN_DELETE|
			unix.IN_CLOSE_WRITE|
			unix.IN_MOVED_TO|
			unix.IN_MOVED_FROM|
			unix.IN_MOVE_SELF,
	)
	if err != nil {
		log.Error().Stack().Err(err).Msg("error inotify on " + dir)
		return
	}
	var buff [(unix.SizeofInotifyEvent + unix.NAME_MAX + 1) * 20]byte

	for {
		offset := 0
		n, err := unix.Read(fd, buff[:])
		if err != nil {
			log.Error().Stack().Err(err).Msg("error inotify on " + dir)
			return
		}

		for offset < n {
			e := (*unix.InotifyEvent)(unsafe.Pointer(&buff[offset]))

			nameBs := buff[offset+unix.SizeofInotifyEvent : offset+unix.SizeofInotifyEvent+int(e.Len)]
			name := string(bytes.TrimRight(nameBs, "\x00"))
			if len(name) > 0 && e.Mask&unix.IN_ISDIR == unix.IN_ISDIR {
				name += " (dir)"
			}

			// HERE
			match, err := regexp.Match(regex, []byte(name))
			if err != nil {
				log.Error().Stack().Err(err).Msg("error in regexp match" + regex)
				return
			}
			if match {
				fmt.Printf("jeff file %s matches the regex\n", name)
			} else {
				fmt.Printf("jeff file %s does NOT match the regex\n", name)
			}

			switch {
			case e.Mask&unix.IN_CREATE == unix.IN_CREATE:
				fmt.Printf("CREATE %v\n", name)
			case e.Mask&unix.IN_DELETE == unix.IN_DELETE:
				fmt.Printf("DELETE %v\n", name)
			case e.Mask&unix.IN_CLOSE_WRITE == unix.IN_CLOSE_WRITE:
				fmt.Printf("CLOSE_WRITE %v\n", dir+"/"+name)
				if match {
					fmt.Printf("adding f " + dir + "/" + name + " to processing queue\n")
					s.QueueOfFiles <- QueueEntry{dirPath: dir, filePath: dir + "/" + name, regex: regex}
				}
			case e.Mask&unix.IN_MOVED_TO == unix.IN_MOVED_TO:
				fmt.Printf("IN_MOVED_TO [%v] %v\n", e.Cookie, name)
				if match {
					fmt.Printf("adding " + dir + "/" + name + " to processing queue\n")
					s.QueueOfFiles <- QueueEntry{dirPath: dir, filePath: dir + "/" + name, regex: regex}
				}
			case e.Mask&unix.IN_MOVED_FROM == unix.IN_MOVED_FROM:
				fmt.Printf("IN_MOVED_FROM [%v] %v\n", e.Cookie, name)
			case e.Mask&unix.IN_MOVE_SELF == unix.IN_MOVE_SELF:
				fmt.Printf("IN_MOVE_SELF %v\n", name)
			}
			offset += int(unix.SizeofInotifyEvent + e.Len)
		}
	}
}
