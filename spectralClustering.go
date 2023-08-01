package main

import (
	"errors"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"log"
	"math"
	kmeans "muesli-kmeans-0ad7a62"
	"reflect"
	"sort"
)

func spectral_cluster(data kmeans.Points){
	similarity := calEuclidDistanceMatrix(data)
	myKnn := knn(similarity,3)
	lm := laplace_matrix(myKnn)
	var eigsym mat.EigenSym
	n := len(lm.RawRowView(0))
	lmSy := mat.NewSymDense(n,lm.RawMatrix().Data)
	ok := eigsym.Factorize(lmSy, true)
	if !ok {
		log.Fatal("Symmetric eigendecomposition failed")
	}
	fmt.Printf("Eigenvalues of A:\n%1.3f\n\n", eigsym.Values(nil))

	var ev mat.Dense
	eigsym.VectorsTo(&ev)
	fmt.Printf("Eigenvectors of A:\n%1.3f\n\n", mat.Formatted(&ev))
	eignVlaluse := eigsym.Values(nil)
	indices := make([]int,n)
	for j:=0;j<n;j++{
		indices[j]=j
	}
	sort.Slice(indices, func(i,j int) bool {return eignVlaluse[indices[i]]<eignVlaluse[indices[j]]})
	sort.Slice(eignVlaluse,func(i,j int) bool {return eignVlaluse[indices[i]]<eignVlaluse[indices[j]]})
	fmt.Println(eignVlaluse, indices) // [dog cat apple bat] [2 1 0 3]
	lastV := 0.0
	index  := 0
	maxV := -1.0
	for i,values := range eignVlaluse {
		if values - lastV > maxV {
			lastV = values
			index = i
		}
	}
	k := index + 1
	k = 2
	eg_data := make([][]float64,n)
	for i := 0;i<n;i++ {
		eg_data[i] = make([]float64,k)
	}
	for i := 0 ;i< n; i++ {
		for j := 0;j< k ;j++ {
			eg_data[i][j] = ev.At(i,j)
		}
	}
	/*var datas []Point
	for i:= 0 ;i<n ;i++ {
		var p Point
		for j :=0;j<k;j++ {
			p.Entry = append(p.Entry, eg_data[i][j])
		}
		datas = append(datas,p)
	}
	clust := KMEANS(datas, uint64(k), 0.001)
	for i,c := range clust {
		fmt.Println(i,c)
	}*/
	var datas kmeans.Points
	var ds kmeans.Points
	for i:= 0 ;i<n ;i++ {
		datas = append(datas,eg_data[i])
		ds = append(ds,eg_data[i])
	}
	fmt.Println(len(ds))
	km := kmeans.New()

	Clus,_ := km.Partition(datas,2)
	cs := make([]kmeans.Points,len(Clus))
	for i,c := range Clus {
		for _,p := range c.Points {
			for j,d := range ds{
				if ReflectEqual(p,d) {
					fmt.Println("!231")
					cs[i] = append(cs[i],data[j])
					break
				}
			}
		}

	}
	for i,c := range cs  {
		fmt.Println(i,c)
	}
}
func ReflectEqual(x, y interface{}) bool {
	return reflect.DeepEqual(x, y)
}

func laplace_matrix(adjacentMatrix *mat.Dense) *mat.Dense{
	N,_ := adjacentMatrix.Dims()
	//计算度矩阵
	degreeMatrix := mat.NewDense(N,N,nil)
	for i := 0 ;i < N ;i++ {
		ri := adjacentMatrix.RawRowView(i)
		sum := 0.0
		for j := 0;j< N ;j++ {
			sum += ri[j]
		}
		degreeMatrix.Set(i,i,sum)
	}
	//未正则拉普拉斯矩阵
	var L1 *mat.Dense
	L1 = degreeMatrix
	L1.Sub(L1,adjacentMatrix)
	var D *mat.Dense
	D = degreeMatrix
	for i := 0;i<N ;i++ {
		x := D.At(i,i)
		x = 1/math.Sqrt(x)
		D.Set(i,i,x)
	}
	var L = new(mat.Dense)
	L.Mul(D,L1)
	L.MulElem(L,D)
	return L
}
func knn(simi *mat.Dense,k int) *mat.Dense{
	_,N := simi.Dims()
	M := mat.NewDense(N,N,nil)
	indices := make([]int,N)
	for i :=0;i<N ;i++ {
		for j:=0;j<N;j++{
			indices[j]=j
		}
		dist_with_index := mat.Row(nil,i,simi)
		sort.Slice(indices, func(k,t int) bool {return dist_with_index[indices[k]]>dist_with_index[indices[t]]})
		fmt.Println(dist_with_index, indices) // [dog cat apple bat] [2 1 0 3]
		for j := 0 ;j< k ;j++{
			M.Set(i,indices[j],dist_with_index[indices[j]])
			M.Set(indices[j],i,dist_with_index[indices[j]])
		}
	}
	return M
}
func calEuclidDistanceMatrix(data kmeans.Points)*mat.Dense{
	X := len(data)
	S := mat.NewDense(X,X,nil)
	for i := 0;i<X;i++ {
		for j := i+1;j<X;j++ {
			simi := euclidDistance(data[i],data[j])
			S.Set(i,j,simi)
			S.Set(j,i,simi)
		}
	}
	return S
}

//后期修改
func euclidDistance(x1,x2 kmeans.Point) float64 {
	res ,_:= CosineSimi(x1,x2)
	return res

	/*var c []float64
	c = append(c, x1)
	c = append(c,x2)
	simi,_ :=  CosineSimi(c,c)
	return simi*/

}
func CosineSimi(s1,s2 kmeans.Point)(ssim float64,err error){
	m := s1[0]*s2[0] + s1[1]*s2[1]
	d := (math.Sqrt(math.Pow(s1[0],2)+ math.Pow(s1[1],2)))*(math.Sqrt(math.Pow(s2[0],2)+ math.Pow(s2[1],2)))
	if d == 0  {
		return 0.0, errors.New("Vectors should not be null (all zeros)")
	}
	return m/d, nil
}