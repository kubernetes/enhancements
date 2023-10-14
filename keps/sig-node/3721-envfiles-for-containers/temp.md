```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-with-shared-env-vars
spec:
  volumes:
  - name: shared-env-volume
    emptyDir: {}
  initContainers:
  - name: init-container-1
    image: busybox:1.28
    command: ['sh', '-c', 'echo "INIT1_MESSAGE=Init container 1 executed" > /shared-env/vars.env']
    volumeMounts:
    - name: shared-env-volume
      mountPath: /shared-env
    envVolumeKeyRef:
      name: INIT1_MESSAGE
      key: vars.env
  - name: init-container-2
    image: busybox:1.28
    command: ['sh', '-c', 'echo "INIT2_MESSAGE=Init container 2 executed" >> /shared-env/vars.env']
    volumeMounts:
    - name: shared-env-volume
      mountPath: /shared-env
    envVolumeKeyRef:
      name: INIT2_MESSAGE
      key: vars.env
  containers:
  - name: main-container
    image: nginx:1.14.2
    ports:
    - containerPort: 80
    volumeMounts:
    - name: shared-env-volume
      mountPath: /shared-env
    envVolumeKeyRef:
      name: SHARED_ENV_VARS
      key: vars.env

```