import json

application = {
    "auth": {
        "chat": 1,
        "posts": 1,
        "db": 1
    },
    "chat": {
        "db": 4
    },
    "posts": {
        "db": 2
    }
}

# application = {
#     "c1": {
#         "c2": 1,
#         "c3": 1
#     },
#     "c2": {
#         "c3": 4
#     }
# }

topology = {
    "n1": {
        "n2": 4
    },
    "n2": {
        "n3": 4,
        "n4": 4
    },
    "n3": {
        "n4": 4,
        "n5": 8
    },
    "n4": {
        "n5": 8,
        "n6": 4
    },
    "n5": {
        "n7": 4
    },
    "n6": {
        "n8": 4,
        "n9": 8
    },
    "n7": {},
    "n8": {},
    "n9": {},
}

# topology = {
#     "n1": {
#         "n2": 5,
#         "n4": 5,
#     },
#     "n2": {
#         "n3": 5
#     },
#     "n3": {
#         "n4": 5
#     }
# }

def fill(inp):
    ret = {}

    for parent, childset in inp.items():
        for child, value in childset.items():
            if parent not in ret:
                ret[parent] = {}
            if child not in ret:
                ret[child] = {}
            ret[parent][child] = value
            ret[child][parent] = value

    return ret

def get_edges(inp):
    ret = []
    
    for parent, childset in inp.items():
        for child, value in childset.items():
            ret.append((value, parent, child))

    ret.sort(reverse=True)
    return ret

# with open("sample.json", "w") as outfile:
#     json.dump(fill(topology), outfile)
