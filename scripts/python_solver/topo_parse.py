import json

def parse_topo(fname):
    f = open(fname)
    data = json.load(f)
    f.close()

    topo = {}

    for node in data:
        topo[node["id"]] = {}

    for node in data:
        if "adjacencies" not in node:
            continue

        for adjacencies in node["adjacencies"]:
            if ("nodeTo" not in adjacencies) or ("data" not in adjacencies):
                continue

            if ("bw" not in adjacencies["data"]) or ("channel" not in adjacencies["data"]):
                continue

            if adjacencies["data"]["channel"].startswith("eth") or (adjacencies["data"]["channel"] in ['br-lan']):
                continue

            topo[node["id"]][adjacencies["nodeTo"]] = adjacencies["data"]["bw"]
            topo[adjacencies["nodeTo"]][node["id"]] = adjacencies["data"]["bw"]

    return topo

print(parse_topo("/Users/abauskar/Workspaces/mesh-bw-scheduler/scripts/python_solver/topo/qmp_2022-11-14_09.json"))
