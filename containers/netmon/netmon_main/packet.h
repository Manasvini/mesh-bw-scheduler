#include <linux/ip.h>

// Returns the protocol byte for an IP packet, 0 for anything else
static __always_inline u64 lookup_protocol(struct xdp_md *ctx)
{
    u64 protocol = 0;

    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    struct ethhdr *eth = data;
    if (data + sizeof(struct ethhdr) > data_end)
        return 0;

    // Check that it's an IP packet
    if (bpf_ntohs(eth->h_proto) == ETH_P_IP)
    {
        // Return the protocol of this packet
        // 1 = ICMP
        // 6 = TCP
        // 17 = UDP        
        struct iphdr *iph = data + sizeof(struct ethhdr);
        if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) <= data_end)
            protocol = iph->protocol;
    }
    return protocol;
}

static inline int parse_ipv4_dest( struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    struct ethhdr *eth = data;
    if (data + sizeof(struct ethhdr) > data_end)
        return 0;
	int daddr = 0;
    // Check that it's an IP packet
    if (bpf_ntohs(eth->h_proto) == ETH_P_IP)
    {
     	struct iphdr *iph = data + sizeof(struct ethhdr);
    	if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) <= data_end)
            daddr = iph->saddr;
	}
	return daddr;
}

static inline int get_pkt_size( struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;
    struct ethhdr *eth = data;
    if (data + sizeof(struct ethhdr) > data_end)
        return 0;
	int size = 0;
    // Check that it's an IP packet
    if (bpf_ntohs(eth->h_proto) == ETH_P_IP)
    {
     	struct iphdr *iph = data + sizeof(struct ethhdr);
    	if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) <= data_end)
            size = ctx->data_end - ctx->data;
	}
	return size;
}
