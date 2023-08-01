from matplotlib import pyplot as plt
from itertools import cycle, islice
from mpl_toolkits.mplot3d import Axes3D
import numpy as np

def plot(X,label, y_sp, y_km):
    colors = np.array(list(islice(cycle(['#377eb8', '#ff7f00', '#4daf4a',
                                                '#f781bf', '#a65628', '#984ea3',
                                                '#999999', '#e41a1c', '#dede00']),
                                        int(max(y_sp) + 1))))
    print(type(y_sp))
   # plt.subplot(121)
    fig = plt.figure()
    ax = fig.gca(projection="3d")
    #Axes3D.scatter(xs, ys, zs=zs1, zdir="z", c="#00DDAA", marker="o", s=40)
   # ax.scatter(X[:,0], X[:,1],X[:,2],zdir="z", c=colors[y_sp], marker="o", s=40)
    #plt.title("Spectral Clustering")

    #plt.subplot(122)
   # plt.scatter(X[:,0], X[:,1], s=10, color=colors[y_km])
   # plt.title("Kmeans Clustering")
    # plt.show()
    ax.scatter(X[:, 0], X[:, 1], X[:, 2], zdir="z", c=colors[y_sp], marker="o", s=10)
    plt.savefig("spectral_clustering1.png")
    #plt.scatter(X[:, 0], X[:, 1], s=10, color=colors[y_km])

    plt.style.use("ggplot")
    plt.figure()
   # plt.title("Data")
    plt.scatter(X[:, 0], X[:, 1], marker="o", c=np.squeeze(label), s=20)
    plt.savefig("3.png")

    #plt.subplot(121)
    plt.scatter(X[:, 0], X[:, 1], s := 10, color := colors[y_sp])
   # plt.title("Spectral Clustering")
   # plt.subplot(122)
    #plt.scatter(X[:, 0], X[:, 1], s := 5, color := colors[y_km])
    #plt.title("Kmeans Clustering")
    # plt.show()
    plt.savefig("spectral_clustering.png")

