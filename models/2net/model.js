/* ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 * 2 node system
 * ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/

// hey! listen! ...............................................................
//
// Raven is a tool for development. As such one of the primary things raven
// does is plop your code in an environment suitable for testing. In this
// system model the following source directories must be provided via
// environment variables so raven knows where to find them and can mount
// them into the development and testing environment.
//
// AGXDIR - The AgentX source code repository
// SWITCHDIR - The switch controller code repository
// NETLINKDIR - The go netlink library
// WALRUSDIR - The walrus testing framework library
// WKDIR - the source dir for this raven system
//
// To automatically fetch all of these repositories and define the associated
// environment variables run the fetchenv.sh script sourced to your current
// shell
// ............................................................................



controller = {
  'name': 'control',
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 4
  },
  'memory': {
    'capacity': GB(4)
  },
  'mounts': [
    { 'source': env.SWITCHDIR, 'point': '/opt/switch-drivers'}
  ]
}

walrus = {
  'name': 'walrus',
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 1
  },
  'memory': {
    'capacity': GB(2)
  },
  'mounts': [
    { 'source': env.WALRUSDIR, 'point': '/opt/walrus'},
    { 'source': env.WKDIR+'/config/files/walrus', 'point': '/tmp/config' }
  ]
}

zwitch = {
  'name': 'nimbus',
  'image': 'cumulusvx-3.5',
  'os': 'linux',
  'mounts': [
    { 'source': env.AGXDIR, 'point': '/opt/agx' },
    { 'source': env.NETLINKDIR, 'point': '/opt/netlink' },
    { 'source': env.SWITCHDIR, 'point': '/opt/switch-drivers'},
    { 'source': env.WKDIR+'/config/files/nimbus', 'point': '/tmp/config' }
  ]
};

nodes = Range(2).map(i => ({
  'name': `n${i}`,
  'image': 'debian-stretch',
  'os': 'linux',
  'cpu': {
    'cores': 4
  },
  'memory': {
    'capacity': GB(6)
  },
  'mounts': [
    { 'source': env.WALRUSDIR, 'point': '/opt/walrus'},
    { 'source': env.WKDIR+'/config/files/node', 'point': '/tmp/config' }
  ]
}));

links = [
  Link('walrus', 1, 'nimbus', 1),
  Link('control', 1, 'nimbus', 2),
  ...Range(2).map(i => Link(`n${i}`, 1, 'nimbus', i+3)),
]

topo = {
  'name': '2net',
  'nodes':[controller, walrus, ...nodes],
  'switches': [zwitch],
  'links': links
};
