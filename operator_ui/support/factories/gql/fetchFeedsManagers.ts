// buildFeedsManager builds a feeds manager for the FetchFeedsManagers query.
export function buildFeedsManager(
  overrides?: Partial<FetchFeedsManagers['feedsManagers']['results'][number]>,
): FetchFeedsManagers['feedsManagers']['results'][number] {
  return {
    __typename: 'FeedsManager',
    id: '1',
    name: 'Chainlink Feeds Manager',
    uri: 'localhost:8080',
    publicKey: '1111',
    jobTypes: ['FLUX_MONITOR'],
    isConnectionActive: false,
    isBootstrapPeer: false,
    bootstrapPeerMultiaddr: null,
    createdAt: new Date(),
    ...overrides,
  }
}
