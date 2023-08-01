# coding=utf-8
import csv
import sys
import random
import time

sys.path.append("..")

from utils.similarity import calEuclidDistanceMatrix
from utils.knn import myKNN
from utils.laplacian import calLaplacianMatrix
from utils.dataloader import genTwoCircles
from utils.ploter import plot
from sklearn.cluster import KMeans
import numpy as np
import pandas as pd

np.random.seed(1)


#data, label = genTwoCircles(n_samples=1000)
data1 = pd.read_csv(r'G:\Go workplace\src\ra\new_data.csv')
data_array = np.array(data1)
data_list =data_array.tolist()

arr = []
for point in data_list:
    arr.append(point[0:3])
data = np.array(arr)
arr2 = []
for point in data_list:
    arr2.extend(point[3:])
for i in range(0,len(arr2)):
    arr2[i] = int(arr2[i])
label = np.array(arr2)

for i in range(0,len(arr)):
   arr[i] = arr[i][:2]

Similarity = calEuclidDistanceMatrix(data)
print(len(Similarity[0]))
Adjacent = myKNN(Similarity, k=10)

Laplacian = calLaplacianMatrix(Adjacent)

x, V = np.linalg.eig(Laplacian)

x = zip(x, range(len(x)))
x = sorted(x, key=lambda x:x[0])
print(x[1][1])
max = 0
index = 0
for i in range(1,10):
    x1 = ((x[i][1] - x[i-1][1])**2 +(x[i][0] - x[i-1][0])**2)** 0.5
    if x1 > max:
        index = i
        max = x1
        print(max)
print(index)
H = np.vstack([V[:,i] for (v, i) in x[:index]]).T

sp_kmeans = KMeans(n_clusters=index).fit(H)
pure_kmeans = KMeans(n_clusters=10).fit(data)

plot(data,label, sp_kmeans.labels_,pure_kmeans.labels_)
list2 = arr
list1 = sp_kmeans.labels_.tolist()
list2.insert(0,['x','y'])
with open(r"G:\Go workplace\src\subjectivelogic\new_data.csv", mode="w", encoding="utf-8-sig", newline="") as f:

    # 基于打开的文件，创建 csv.writer 实例
    writer = csv.writer(f)

    # 写入 header。
    # writerow() 一次只能写入一行。

    # 写入数据。
    # writerows() 一次写入多行。
    writer.writerows(list2)

data = pd.read_csv(r'G:\Go workplace\src\ra\new_data.csv')
data['label'] = list1
data.to_csv(r"G:\Go workplace\src\ra\new_data.csv",index =False)
