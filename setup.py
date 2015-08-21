#!/usr/bin/python
from optparse import OptionParser
from optparse import OptionGroup
import commands
import subprocess
import time
import datetime
import os

class NSLog:
  """ Netscaler Log Class. """
  def __init__(self, logfile="./nslog"):
    self.logfile = logfile
    self.fh = open(logfile, "a")

  def log(self, message):
    self.fh.write(str(datetime.datetime.now()) + " " + message + "\n")

class NSImage:
    """ Netscaler Image Class. """
    def __init__(self, id, image="nme"):
        self.image = image
        self.pid = 0
        self.containerid = 0
        self.numintf = 0
        self.nsip = 0
        self.nsloopip = 0
        self.host_sshport = "2222"
        self.host_httpport = "8080"
        self.id = id
        self.hostip = ""
        self.cip = "" 
        self.dockerip = 0

    def get_container_id(self):
        print "Running image " + self.image
        (status, output) = commands.getstatusoutput("/usr/bin/docker ps -q -f status=running -f name=NetScalerME")
        self.containerid = output
        self.nslfh.write(output)

    def get_container_pid(self):
        (status, output) = commands.getstatusoutput("/usr/bin/docker inspect -f {{.State.Pid}} " +  self.containerid)
        self.pid = output
        print "Started container cid=" + self.containerid +" pid=" + self.pid
        nslog.log("Netscaler Container containerid: " + output + " pid=" + self.pid)
        (status, output) = commands.getstatusoutput("mkdir -p /var/run/netns")
        (status, output) = commands.getstatusoutput("ln -s /proc/" + self.pid + "/ns/net /var/run/netns/" + self.pid)

    def set_host_sshport(self, port):
         self.host_sshport = port


    def set_host_httpport(self, port):
         self.host_httpport = port

    def set_controller_ip(self, cip):
        self.cip = cip

    def set_host_ip(self, hostip):
        self.hostip = hostip

    def set_nsloop_ip(self, nsloopip):
        self.nsloopip = nsloopip

    def set_nsip(self, nsip):
        self.nsip = nsip

    def append_nsip_conf(self):

        argstring = ""
	if len(self.hostip):
          argstring += '-o "' + self.hostip +":" + self.host_sshport + ":" + self.host_httpport + '"'
        argstring += ' -l "' + self.dockerip + '" -m "' + self.nsip + '"'
        if len(self.cip):
          argstring += ' -c "'  + self.cip + '"'
        argstring += ' -d "' + self.dockerip + '"'
        print argstring

        (status, output) = commands.getstatusoutput("/usr/bin/docker exec " +  self.containerid + ' /var/netscaler/bins/conf.py ' + argstring)


    def create_interface(self, intf_name, bridge):
        hostintf = "BR" +  self.pid + "_" + str(self.numintf) 
        nsintf = "NS" +  self.pid + "_" + str(self.numintf) 
        (status, output) = commands.getstatusoutput("ip link add " + nsintf + " type veth peer name " + hostintf + " && echo ok")
        print output
        (status, output) = commands.getstatusoutput("ip link set " + nsintf +" netns "+ self.pid + " && echo ok")
        print output
        (status, output) = commands.getstatusoutput("ip netns exec " + self.pid + " ip link set dev " + nsintf + " name " + intf_name + " && echo ok")
        print output
        (status, output) = commands.getstatusoutput("ip netns exec " + self.pid + " ip link set " + intf_name + " up && echo ok")
        print output
        (status, output) = commands.getstatusoutput("/sbin/brctl addif " + bridge + " " + hostintf + " && echo ok")
        print output
        (status, output) = commands.getstatusoutput("ip link set " + hostintf + " up && echo ok")
        print output
        self.numintf += 1

    def create_link_local(self):
        (status, output) = commands.getstatusoutput("nsenter -n -t " + self.pid + " ip addr add 169.254.1.100/16 dev eth0 && echo ok")
        print output
        (status, output) = commands.getstatusoutput("ip addr add 169.254.1.200/16 dev docker0 && echo ok")
        print output

    def run(self, bridge):
        "Runs a netscaler image and attaches to the given bridge."
	try:
            self.kill()
	except:
	    pass
	
        self.nslfh = open(".nslock" + self.id, "w")
        nslog.log("Going to run netscaler image " + self.image)
        #subprocess.Popen(["/usr/bin/docker", "stop", self.id])
        #subprocess.Popen(["/usr/bin/docker", "rm", "-f", self.id])
        subprocess.Popen(["/usr/bin/docker", "run", "--name", "NetScalerME", "--privileged", "-l", "io.rancher.container.network=true", "--hostname", self.id, "-p", self.host_sshport + ":22",  "-p" , self.host_httpport + ":80", "-d", "-t",  self.image, "/bin/bash" ])
        time.sleep(5)
        self.get_container_id()
        self.get_container_pid()
        self.create_interface("eth1", bridge)
	self.create_link_local()
        print "Linked Netscaler container cid=" + self.containerid +" pid=" + self.pid + " to bridge " + bridge
        (status, output) = commands.getstatusoutput("/usr/bin/docker inspect -f {{.NetworkSettings.IPAddress}} " + self.containerid)
        self.dockerip = output
        subprocess.Popen(["/usr/bin/nme-controller", "-nme", self.dockerip, "-nmeContainerId", self.containerid])
        self.append_nsip_conf()
        nslog.log("Linked Netscaler Container containerid: " + self.containerid + " pid=" + self.pid + " to bridge " + bridge)
        nslog.log("Netscaler Started with ip =" + self.dockerip)

    def kill(self):
        "Kills the current running netscaler container."
        if os.path.isfile(".nslock" + self.id):
            f = open(".nslock" + self.id, "r")
            containerid = f.readline()
            nslog.log("Killing Netscaler Container " + containerid)
            (status, output) = commands.getstatusoutput("/usr/bin/docker inspect -f {{.State.Pid}} " + containerid)
            self.pid = output
            commands.getstatusoutput("/usr/bin/docker stop " + containerid)
	    try:
                os.remove(".nslock" + self.id)
	    except:
		pass
	    try:
                s.remove("/var/run/netns/" + self.pid)
	    except:
		pass

def run_netscaler(options):
  nsi = NSImage(options.key, options.run)
  
  nsi.set_nsip(options.nsip)
  if options.sshport:
      nsi.set_host_sshport(options.sshport)
      print "setting ssh port to:" + options.sshport
  if options.httpport:
      nsi.set_host_httpport(options.httpport)
      print "setting http port to:" + options.httpport
  nsi.run("docker0")

########## Add all the arguments here ##########
parser = OptionParser()
parser.add_option("-r", "--run", dest="run", metavar="<imagename>",
                  help="Download and run the Netscaler Docker image.",
                  default=False)
parser.add_option("-k", "--key", dest="key", metavar="<netscalerid>",
                  help="Netscaler bootup key value")
parser.add_option("", "--nsipmask", dest="nsip", metavar="<nsip netmask>",
                  help="netscaler ip interface netmask",
                  default=False)
parser.add_option("", "--nssshport", dest="sshport", metavar="<port>",
                  help=" netscaler ssh port exposed to host",
                  default=False)
parser.add_option("", "--nshttpport", dest="httpport", metavar="<port>",
                  help="netscaler http port exposed to host",
                  default=False)

########## Parser arguments ends here ##########

nslog = NSLog("./ns.log")
(options, args) = parser.parse_args()
if options.run:
  if options.key is None:
    raise "Key is not given"
  if options.nsip is None:
    raise "NSIP not given"
  run_netscaler(options)

