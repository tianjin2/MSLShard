from sklearn import datasets

def genTwoCircles(n_samples=1000):
    X, y = datasets.make_blobs(n_samples, n_features=2, centers=10 , cluster_std=1.5, random_state=1)

    return X, y

