module github.com/GoogleContainerTools/kpt/porch

go 1.21

replace (
	github.com/GoogleContainerTools/kpt => ../
	github.com/GoogleContainerTools/kpt/porch/api => ./api
	github.com/go-git/go-git/v5 v5.4.3-0.20220408232334-4f916225cb2f => github.com/platkrm/go-git/v5 v5.4.3-0.20220410165046-c76b262044ce
)
