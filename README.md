An ansible-navigator like OpenShift navigator

![screenshot](sss/oc-navigator.png)

```bash
go get github.com/gdamore/tcell/v2
go get github.com/rivo/tview

go run main.go

# or

go build -o oc-navigator
```

Make sure you do -

```
oc login --token=sha256~TESTERTERTETERETETETETETETETET --server=https://api.openshiftapps.com
```
